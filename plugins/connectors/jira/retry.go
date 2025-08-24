/* Copyright © INFINI LTD. All rights reserved.  
 * Web: https://infinilabs.com  
 * Email: hello#infini.ltd */  
  
 package jira  
  
 import (  
	 "context"  
	 "fmt"  
	 "math"  
	 "time"  
   
	 log "github.com/cihub/seelog"  
 )  
   
 // RetryConfig 重试配置  
 type RetryConfig struct {  
	 MaxRetries      int  
	 InitialInterval time.Duration  
	 MaxInterval     time.Duration  
	 Multiplier      float64  
 }  
   
 // DefaultRetryConfig 默认重试配置  
 func DefaultRetryConfig() *RetryConfig {  
	 return &RetryConfig{  
		 MaxRetries:      3,  
		 InitialInterval: time.Second,  
		 MaxInterval:     30 * time.Second,  
		 Multiplier:      2.0,  
	 }  
 }  
   
 // RetryableFunc 可重试的函数类型  
 type RetryableFunc func() error  
   
 // WithRetry 执行带重试的函数  
 func WithRetry(ctx context.Context, config *RetryConfig, fn RetryableFunc) error {  
	 if config == nil {  
		 config = DefaultRetryConfig()  
	 }  
   
	 var lastErr error  
	 interval := config.InitialInterval  
   
	 for attempt := 0; attempt <= config.MaxRetries; attempt++ {  
		 if attempt > 0 {  
			 log.Debugf("[jira retry] attempt %d/%d after %v", attempt, config.MaxRetries, interval)  
   
			 select {  
			 case <-ctx.Done():  
				 return ctx.Err()  
			 case <-time.After(interval):  
			 }  
   
			 // 计算下次重试间隔（指数退避）  
			 interval = time.Duration(float64(interval) * config.Multiplier)  
			 if interval > config.MaxInterval {  
				 interval = config.MaxInterval  
			 }  
		 }  
   
		 err := fn()  
		 if err == nil {  
			 if attempt > 0 {  
				 log.Infof("[jira retry] succeeded after %d attempts", attempt+1)  
			 }  
			 return nil  
		 }  
   
		 lastErr = err  
   
		 // 检查是否应该重试  
		 if !shouldRetryError(err) {  
			 log.Debugf("[jira retry] error is not retryable: %v", err)  
			 break  
		 }  
   
		 if attempt < config.MaxRetries {  
			 log.Warnf("[jira retry] attempt %d failed, will retry: %v", attempt+1, err)  
		 }  
	 }  
   
	 return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)  
 }  
   
 // shouldRetryError 判断错误是否可重试  
 func shouldRetryError(err error) bool {  
	 if jiraErr, ok := err.(*JiraError); ok {  
		 return jiraErr.Retryable  
	 }  
   
	 // 其他类型的错误，如网络错误等  
	 return true  
 }  
   
 // ExponentialBackoff 计算指数退避时间  
 func ExponentialBackoff(attempt int, baseInterval time.Duration, maxInterval time.Duration) time.Duration {  
	 if attempt <= 0 {  
		 return baseInterval  
	 }  
   
	 backoff := time.Duration(float64(baseInterval) * math.Pow(2, float64(attempt-1)))  
	 if backoff > maxInterval {  
		 backoff = maxInterval  
	 }  
   
	 return backoff  
 }