package proxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CacheManager 缓存管理器接口
type CacheManager interface {
	// GetVersionPath 获取缓存的版本路径
	GetVersionPath(toolName, version string) (string, bool)

	// SetVersionPath 设置版本路径缓存
	SetVersionPath(toolName, version, path string)

	// GetExecutablePath 获取缓存的可执行文件路径
	GetExecutablePath(toolName, version string) (string, bool)

	// SetExecutablePath 设置可执行文件路径缓存
	SetExecutablePath(toolName, version, path string)

	// GetProjectContext 获取缓存的项目上下文
	GetProjectContext(projectPath string) (*ProjectContext, bool)

	// SetProjectContext 设置项目上下文缓存
	SetProjectContext(projectPath string, context *ProjectContext)

	// InvalidateCache 使缓存失效
	InvalidateCache(key string)

	// ClearAll 清除所有缓存
	ClearAll()

	// GetStats 获取缓存统计信息
	GetStats() *CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Evictions int64   `json:"evictions"`
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	HitRatio  float64 `json:"hit_ratio"`
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Value       interface{}
	CreatedAt   time.Time
	AccessedAt  time.Time
	TTL         time.Duration
	AccessCount int64
}

// DefaultCacheManager 默认缓存管理器实现
type DefaultCacheManager struct {
	mu         sync.RWMutex
	cache      map[string]*CacheEntry
	maxSize    int
	defaultTTL time.Duration
	logger     *logrus.Logger

	// 统计信息
	hits      int64
	misses    int64
	evictions int64
}

// NewCacheManager 创建新的缓存管理器
func NewCacheManager(maxSize int, defaultTTL time.Duration) CacheManager {
	return &DefaultCacheManager{
		cache:      make(map[string]*CacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		logger:     logrus.New(),
	}
}

// GetVersionPath 获取缓存的版本路径
func (cm *DefaultCacheManager) GetVersionPath(toolName, version string) (string, bool) {
	key := fmt.Sprintf("version_path:%s:%s", toolName, version)
	if value, ok := cm.get(key); ok {
		if path, ok := value.(string); ok {
			return path, true
		}
	}
	return "", false
}

// SetVersionPath 设置版本路径缓存
func (cm *DefaultCacheManager) SetVersionPath(toolName, version, path string) {
	key := fmt.Sprintf("version_path:%s:%s", toolName, version)
	cm.set(key, path, cm.defaultTTL)
}

// GetExecutablePath 获取缓存的可执行文件路径
func (cm *DefaultCacheManager) GetExecutablePath(toolName, version string) (string, bool) {
	key := fmt.Sprintf("executable_path:%s:%s", toolName, version)
	if value, ok := cm.get(key); ok {
		if path, ok := value.(string); ok {
			return path, true
		}
	}
	return "", false
}

// SetExecutablePath 设置可执行文件路径缓存
func (cm *DefaultCacheManager) SetExecutablePath(toolName, version, path string) {
	key := fmt.Sprintf("executable_path:%s:%s", toolName, version)
	cm.set(key, path, cm.defaultTTL)
}

// GetProjectContext 获取缓存的项目上下文
func (cm *DefaultCacheManager) GetProjectContext(projectPath string) (*ProjectContext, bool) {
	key := fmt.Sprintf("project_context:%s", projectPath)
	if value, ok := cm.get(key); ok {
		if context, ok := value.(*ProjectContext); ok {
			return context, true
		}
	}
	return nil, false
}

// SetProjectContext 设置项目上下文缓存
func (cm *DefaultCacheManager) SetProjectContext(projectPath string, context *ProjectContext) {
	key := fmt.Sprintf("project_context:%s", projectPath)
	cm.set(key, context, cm.defaultTTL)
}

// InvalidateCache 使缓存失效
func (cm *DefaultCacheManager) InvalidateCache(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.cache[key]; exists {
		delete(cm.cache, key)
		cm.evictions++
	}
}

// ClearAll 清除所有缓存
func (cm *DefaultCacheManager) ClearAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	evicted := len(cm.cache)
	cm.cache = make(map[string]*CacheEntry)
	cm.evictions += int64(evicted)

	cm.logger.Infof("Cleared all cache entries: %d", evicted)
}

// GetStats 获取缓存统计信息
func (cm *DefaultCacheManager) GetStats() *CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	total := cm.hits + cm.misses
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(cm.hits) / float64(total)
	}

	return &CacheStats{
		Hits:      cm.hits,
		Misses:    cm.misses,
		Evictions: cm.evictions,
		Size:      len(cm.cache),
		MaxSize:   cm.maxSize,
		HitRatio:  hitRatio,
	}
}

// get 获取缓存值
func (cm *DefaultCacheManager) get(key string) (interface{}, bool) {
	cm.mu.RLock()
	entry, exists := cm.cache[key]
	cm.mu.RUnlock()

	if !exists {
		cm.recordMiss()
		return nil, false
	}

	// 检查是否过期
	if cm.isExpired(entry) {
		cm.InvalidateCache(key)
		cm.recordMiss()
		return nil, false
	}

	// 更新访问时间和计数
	cm.mu.Lock()
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	cm.mu.Unlock()

	cm.recordHit()
	return entry.Value, true
}

// set 设置缓存值
func (cm *DefaultCacheManager) set(key string, value interface{}, ttl time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查容量限制
	if len(cm.cache) >= cm.maxSize {
		cm.evictLRU()
	}

	now := time.Now()
	cm.cache[key] = &CacheEntry{
		Value:       value,
		CreatedAt:   now,
		AccessedAt:  now,
		TTL:         ttl,
		AccessCount: 1,
	}
}

// isExpired 检查缓存条目是否过期
func (cm *DefaultCacheManager) isExpired(entry *CacheEntry) bool {
	if entry.TTL <= 0 {
		return false // 永不过期
	}
	return time.Since(entry.CreatedAt) > entry.TTL
}

// evictLRU 驱逐最近最少使用的条目
func (cm *DefaultCacheManager) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range cm.cache {
		if oldestKey == "" || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(cm.cache, oldestKey)
		cm.evictions++
	}
}

// recordHit 记录缓存命中
func (cm *DefaultCacheManager) recordHit() {
	cm.mu.Lock()
	cm.hits++
	cm.mu.Unlock()
}

// recordMiss 记录缓存未命中
func (cm *DefaultCacheManager) recordMiss() {
	cm.mu.Lock()
	cm.misses++
	cm.mu.Unlock()
}

// FastPathResolver 快速路径解析器
type FastPathResolver struct {
	cache     CacheManager
	logger    *logrus.Logger
	pathCache map[string]string // toolName -> path
	mu        sync.RWMutex
}

// NewFastPathResolver 创建快速路径解析器
func NewFastPathResolver(cache CacheManager) *FastPathResolver {
	return &FastPathResolver{
		cache:     cache,
		logger:    logrus.New(),
		pathCache: make(map[string]string),
	}
}

// ResolveFast 快速解析工具路径
func (fpr *FastPathResolver) ResolveFast(toolName, version string) (string, bool) {
	// 先检查内存缓存
	fpr.mu.RLock()
	cacheKey := fmt.Sprintf("%s@%s", toolName, version)
	if path, exists := fpr.pathCache[cacheKey]; exists {
		fpr.mu.RUnlock()
		return path, true
	}
	fpr.mu.RUnlock()

	// 检查持久缓存
	if path, ok := fpr.cache.GetExecutablePath(toolName, version); ok {
		// 更新内存缓存
		fpr.mu.Lock()
		fpr.pathCache[cacheKey] = path
		fpr.mu.Unlock()
		return path, true
	}

	return "", false
}

// SetFast 快速设置工具路径
func (fpr *FastPathResolver) SetFast(toolName, version, path string) {
	cacheKey := fmt.Sprintf("%s@%s", toolName, version)

	// 更新内存缓存
	fpr.mu.Lock()
	fpr.pathCache[cacheKey] = path
	fpr.mu.Unlock()

	// 更新持久缓存
	fpr.cache.SetExecutablePath(toolName, version, path)
}

// LazyLoader 延迟加载器
type LazyLoader struct {
	mu      sync.RWMutex
	loaders map[string]func() (interface{}, error)
	cache   map[string]interface{}
	logger  *logrus.Logger
}

// NewLazyLoader 创建延迟加载器
func NewLazyLoader() *LazyLoader {
	return &LazyLoader{
		loaders: make(map[string]func() (interface{}, error)),
		cache:   make(map[string]interface{}),
		logger:  logrus.New(),
	}
}

// Register 注册延迟加载函数
func (ll *LazyLoader) Register(key string, loader func() (interface{}, error)) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	ll.loaders[key] = loader
}

// Load 延迟加载数据
func (ll *LazyLoader) Load(ctx context.Context, key string) (interface{}, error) {
	// 检查缓存
	ll.mu.RLock()
	if value, exists := ll.cache[key]; exists {
		ll.mu.RUnlock()
		return value, nil
	}

	loader, exists := ll.loaders[key]
	ll.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no loader registered for key: %s", key)
	}

	// 加载数据
	value, err := loader()
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", key, err)
	}

	// 缓存结果
	ll.mu.Lock()
	ll.cache[key] = value
	ll.mu.Unlock()

	return value, nil
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	mu      sync.RWMutex
	metrics map[string]*PerformanceMetric
	logger  *logrus.Logger
}

// PerformanceMetric 性能指标
type PerformanceMetric struct {
	Name         string        `json:"name"`
	Count        int64         `json:"count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	MinTime      time.Duration `json:"min_time"`
	MaxTime      time.Duration `json:"max_time"`
	LastExecuted time.Time     `json:"last_executed"`
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*PerformanceMetric),
		logger:  logrus.New(),
	}
}

// StartTimer 开始计时
func (pm *PerformanceMonitor) StartTimer(name string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		pm.RecordMetric(name, duration)
	}
}

// RecordMetric 记录性能指标
func (pm *PerformanceMonitor) RecordMetric(name string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metric, exists := pm.metrics[name]
	if !exists {
		metric = &PerformanceMetric{
			Name:    name,
			MinTime: duration,
			MaxTime: duration,
		}
		pm.metrics[name] = metric
	}

	metric.Count++
	metric.TotalTime += duration
	metric.AverageTime = metric.TotalTime / time.Duration(metric.Count)
	metric.LastExecuted = time.Now()

	if duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}
}

// GetMetrics 获取所有性能指标
func (pm *PerformanceMonitor) GetMetrics() map[string]*PerformanceMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*PerformanceMetric)
	for name, metric := range pm.metrics {
		// 创建副本
		metricCopy := *metric
		result[name] = &metricCopy
	}
	return result
}

// GetMetric 获取指定性能指标
func (pm *PerformanceMonitor) GetMetric(name string) (*PerformanceMetric, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metric, exists := pm.metrics[name]
	if exists {
		metricCopy := *metric
		return &metricCopy, true
	}
	return nil, false
}

// ClearMetrics 清除所有性能指标
func (pm *PerformanceMonitor) ClearMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.metrics = make(map[string]*PerformanceMetric)
}

// OptimizedProxy 优化的代理实现
type OptimizedProxy struct {
	*DefaultCommandProxy
	cache            CacheManager
	fastPathResolver *FastPathResolver
	lazyLoader       *LazyLoader
	perfMonitor      *PerformanceMonitor
}

// NewOptimizedProxy 创建优化的代理
func NewOptimizedProxy(baseProxy *DefaultCommandProxy) *OptimizedProxy {
	cache := NewCacheManager(1000, 10*time.Minute)
	fastPathResolver := NewFastPathResolver(cache)
	lazyLoader := NewLazyLoader()
	perfMonitor := NewPerformanceMonitor()

	return &OptimizedProxy{
		DefaultCommandProxy: baseProxy,
		cache:               cache,
		fastPathResolver:    fastPathResolver,
		lazyLoader:          lazyLoader,
		perfMonitor:         perfMonitor,
	}
}

// InterceptCommand 优化的命令拦截
func (op *OptimizedProxy) InterceptCommand(cmd string, args []string) error {
	defer op.perfMonitor.StartTimer("intercept_command")()

	// 使用基础实现
	return op.DefaultCommandProxy.InterceptCommand(cmd, args)
}

// GetPerformanceStats 获取性能统计
func (op *OptimizedProxy) GetPerformanceStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_stats":         op.cache.GetStats(),
		"performance_metrics": op.perfMonitor.GetMetrics(),
	}
}
