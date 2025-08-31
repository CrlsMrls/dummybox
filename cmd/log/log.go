package log

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogParams holds parameters for the log endpoint.
type LogParams struct {
	Level       string `json:"level"`       // info, warning, error, random
	Size        string `json:"size"`        // short, medium, long, random
	Message     string `json:"message"`     // specific message (takes precedence over size)
	Interval    int    `json:"interval"`    // seconds between logs, 0 means once
	Duration    int    `json:"duration"`    // total duration in seconds to log messages, 0 means indefinitely
	Correlation string `json:"correlation"` // "false" to exclude correlation ID, anything else includes it
}

// LogHandler generates log messages based on the provided parameters.
func LogHandler(w http.ResponseWriter, r *http.Request) {
	params := LogParams{
		Level:       "info",  // Default level
		Size:        "short", // Default size
		Message:     "",      // Default empty message
		Interval:    0,       // Default: log once
		Duration:    0,       // Default: no duration limit
		Correlation: "true",  // Default: include correlation ID
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		if level := r.URL.Query().Get("level"); level != "" {
			params.Level = level
		}
		if size := r.URL.Query().Get("size"); size != "" {
			params.Size = size
		}
		if message := r.URL.Query().Get("message"); message != "" {
			// URL decode the message
			decoded, err := url.QueryUnescape(message)
			if err == nil {
				params.Message = decoded
			} else {
				params.Message = message
			}
		}
		if intervalStr := r.URL.Query().Get("interval"); intervalStr != "" {
			if i, err := strconv.Atoi(intervalStr); err == nil {
				params.Interval = i
			}
		}
		if durationStr := r.URL.Query().Get("duration"); durationStr != "" {
			if d, err := strconv.Atoi(durationStr); err == nil {
				params.Duration = d
			}
		}
		if correlation := r.URL.Query().Get("correlation"); correlation != "" {
			params.Correlation = correlation
		}
	} else if r.Method == http.MethodPost {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to decode log parameters from JSON body")
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	}

	// Validate parameters
	if !isValidLevel(params.Level) {
		log.Ctx(r.Context()).Warn().Str("level", params.Level).Msg("invalid log level, defaulting to info")
		params.Level = "info"
	}
	if !isValidSize(params.Size) {
		log.Ctx(r.Context()).Warn().Str("size", params.Size).Msg("invalid log size, defaulting to short")
		params.Size = "short"
	}
	if params.Interval < 0 || params.Interval > 3600 { // Max 1 hour interval
		log.Ctx(r.Context()).Warn().Int("interval", params.Interval).Msg("invalid interval, defaulting to 0")
		params.Interval = 0
	}
	if params.Duration < 0 || params.Duration > 86400 { // Max 24 hours duration
		log.Ctx(r.Context()).Warn().Int("duration", params.Duration).Msg("invalid duration, defaulting to 0")
		params.Duration = 0
	}

	log.Ctx(r.Context()).Info().
		Str("level", params.Level).
		Str("size", params.Size).
		Str("message", params.Message).
		Int("interval", params.Interval).
		Int("duration", params.Duration).
		Str("correlation", params.Correlation).
		Msg("log generation request received")

	// Create contexts for different purposes
	var immediateCtx context.Context   // For immediate log entry (uses request context)
	var backgroundCtx context.Context  // For background logging (independent context)
	
	if params.Correlation == "false" {
		immediateCtx = context.Background()
		backgroundCtx = context.Background()
	} else {
		immediateCtx = r.Context()
		// Create background context with correlation ID but independent of HTTP request
		backgroundCtx = context.Background()
		// Copy the logger with correlation ID from request context
		if requestLogger := log.Ctx(r.Context()); requestLogger != nil {
			backgroundCtx = requestLogger.WithContext(backgroundCtx)
		}
	}

	// Generate logs based on interval and duration
	if params.Interval == 0 && params.Duration == 0 {
		// Log once immediately
		level := getActualLevel(params.Level)
		message := getActualMessage(params.Message, params.Size)
		generateLogEntry(immediateCtx, level, message)
	} else {
		// Start background logging using independent context
		go func() {
			var ticker *time.Ticker
			var durationTimer *time.Timer
			
			if params.Interval > 0 {
				ticker = time.NewTicker(time.Duration(params.Interval) * time.Second)
				defer ticker.Stop()
			}
			
			if params.Duration > 0 {
				durationTimer = time.NewTimer(time.Duration(params.Duration) * time.Second)
				defer durationTimer.Stop()
			}

			// Log immediately first
			level := getActualLevel(params.Level)
			message := getActualMessage(params.Message, params.Size)
			generateLogEntry(backgroundCtx, level, message)

			// If no interval, we're done
			if params.Interval == 0 {
				return
			}

			// Continue logging at intervals
			for {
				select {
				case <-ticker.C:
					level := getActualLevel(params.Level)
					message := getActualMessage(params.Message, params.Size)
					generateLogEntry(backgroundCtx, level, message)
				case <-durationTimer.C:
					// Duration expired, stop logging
					if params.Duration > 0 {
						return
					}
				}
			}
		}()
	}

	// Return response
	responseMessage := getActualMessage(params.Message, params.Size)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"level":       params.Level,
		"size":        params.Size,
		"message":     responseMessage,
		"interval":    params.Interval,
		"duration":    params.Duration,
		"correlation": params.Correlation,
		"status":      "log generation started",
	})
}

// isValidLevel checks if the log level is valid.
func isValidLevel(level string) bool {
	validLevels := []string{"info", "warning", "error", "random"}
	for _, validLevel := range validLevels {
		if strings.ToLower(level) == validLevel {
			return true
		}
	}
	return false
}

// isValidSize checks if the log size is valid.
func isValidSize(size string) bool {
	validSizes := []string{"short", "medium", "long", "random"}
	for _, validSize := range validSizes {
		if strings.ToLower(size) == validSize {
			return true
		}
	}
	return false
}

// generateLogMessage creates a random log message based on size.
func generateLogMessage(size string) string {
	shortMessages := []string{
		"System operational (Fake message)",
		"Task completed (Fake message)",
		"Connection established (Fake message)",
		"Process started (Fake message)",
		"Data received (Fake message)",
		"Cache updated (Fake message)",
		"Request processed (Fake message)",
		"Service healthy (Fake message)",
		"User logged in (Fake message)",
		"File uploaded (Fake message)",
		"Backup completed (Fake message)",
		"Server restarted (Fake message)",
		"Config reloaded (Fake message)",
		"Queue emptied (Fake message)",
		"License validated (Fake message)",
		"Heartbeat sent (Fake message)",
		"Token refreshed (Fake message)",
		"Session created (Fake message)",
		"Metrics exported (Fake message)",
		"Alert cleared (Fake message)",
		"Job scheduled (Fake message)",
		"Route updated (Fake message)",
		"Certificate renewed (Fake message)",
		"Database synced (Fake message)",
		"Pipeline triggered (Fake message)",
	}

	mediumMessages := []string{
		"User authentication successful for session ABC123, redirecting to dashboard (Fake message)",
		"Database connection pool initialized with 10 connections, ready to serve requests (Fake message)",
		"Configuration loaded from environment variables, server starting on port 8080 (Fake message)",
		"Background job queue processed 25 items in 150ms, queue size now at 5 (Fake message)",
		"API rate limiting enabled, current threshold set to 100 requests per minute (Fake message)",
		"Memory usage at 45%, garbage collection triggered, freed 25MB of memory (Fake message)",
		"WebSocket connection established with client, real-time updates enabled (Fake message)",
		"Email notification sent successfully to user@example.com regarding order #12345 (Fake message)",
		"Load balancer health check passed, all 8 backend servers responding normally (Fake message)",
		"SSL certificate expiring in 30 days, automatic renewal process initiated (Fake message)",
		"Distributed cache warming completed, 50,000 keys preloaded in 2.3 seconds (Fake message)",
		"OAuth token validation successful, user permissions loaded from directory service (Fake message)",
		"File system cleanup removed 1.2GB of temporary files, disk space recovered (Fake message)",
		"Microservice deployment rollout completed, version 2.1.4 now serving 100% traffic (Fake message)",
		"Database index optimization finished, query performance improved by 35% average (Fake message)",
		"Message broker processed 10,000 events, all consumers keeping up with queue (Fake message)",
		"Container orchestrator scaled down 3 replicas, resource utilization optimized (Fake message)",
		"Monitoring alert threshold adjusted, reducing false positives by 60% (Fake message)",
		"Backup verification completed successfully, all 500GB data integrity confirmed (Fake message)",
		"Network latency to external API improved, average response time now 150ms (Fake message)",
		"User session store migrated to Redis cluster, improved response times observed (Fake message)",
		"Feature flag toggled for premium users, A/B testing framework activated (Fake message)",
		"Log aggregation pipeline processed 2M entries, search indices updated (Fake message)",
		"Security scan completed, no vulnerabilities detected in current deployment (Fake message)",
		"Auto-scaling policy triggered, adding 2 instances due to increased CPU usage (Fake message)",
	}

	longMessages := []string{
		"System performance analysis completed: CPU usage averaged 35% over the last hour with peaks reaching 78% during batch processing operations. Memory consumption remained stable at 2.1GB with minimal garbage collection overhead. Network I/O showed consistent patterns with 450MB inbound and 320MB outbound traffic. Database query performance metrics indicate average response time of 45ms with 99th percentile at 180ms. (Fake message)",
		"User session management update: Successfully migrated 15,000 active user sessions from Redis cluster node-1 to node-2 during scheduled maintenance window. Zero session data loss reported. Session timeout policies updated to extend idle timeout from 30 minutes to 60 minutes for improved user experience. Session cleanup process removed 2,847 expired sessions freeing up 180MB of memory. (Fake message)",
		"Microservices health check results: All 12 services reporting healthy status. API Gateway processed 145,000 requests in the last 10 minutes with 99.97% success rate. Service mesh configuration updated to implement circuit breaker pattern with 50% failure threshold and 30-second timeout. Load balancer distributed traffic evenly across 6 backend instances with average response time of 120ms. (Fake message)",
		"Data pipeline execution summary: ETL process extracted 2.3 million records from source systems, applied 47 transformation rules including data validation, normalization, and enrichment. Data quality checks identified and quarantined 156 records for manual review. Successfully loaded 2,299,844 records into data warehouse. Pipeline completed in 42 minutes, 15% faster than previous run due to query optimization. (Fake message)",
		"Container orchestration deployment analysis: Kubernetes cluster successfully deployed 45 new pods across 3 availability zones with zero downtime. Resource allocation shows 65% CPU utilization and 70% memory usage across all nodes. Persistent volume claims automatically provisioned 2.5TB of storage. Network policies updated to enhance security between namespaces. Ingress controller handling 50,000 requests per minute with SSL termination. (Fake message)",
		"Security audit and compliance report: Comprehensive security scan completed across all infrastructure components. 15 high-priority patches applied to operating systems and runtime environments. Access control policies reviewed and updated for 200 user accounts. Multi-factor authentication compliance achieved 95% adoption rate. Vulnerability assessment identified and remediated 3 medium-risk issues. Security event correlation system processed 100,000 events with 2 actionable alerts. (Fake message)",
		"Database optimization and maintenance cycle: Primary database cluster underwent maintenance window with automatic failover to secondary replica. Index rebuilding process completed for 25 tables, improving query performance by average 40%. Database statistics updated and query planner optimized. Connection pooling configuration tuned for peak load handling. Backup verification confirmed all 500GB of data successfully archived to remote storage. Transaction log cleanup recovered 50GB of disk space. (Fake message)",
		"Machine learning model deployment and monitoring: New recommendation engine model v3.2 deployed to production serving 10,000 users per minute. Model accuracy metrics show 12% improvement over previous version. Feature engineering pipeline processed 50 million data points in batch mode. A/B testing framework configured to gradually increase traffic to 25% of user base. Model inference latency maintained under 50ms p95. Automated retraining pipeline scheduled for weekly execution based on data drift detection. (Fake message)",
		"Network infrastructure upgrade completion: Core network switches upgraded to support 100Gbps backbone connectivity. Software-defined networking policies updated across 15 data centers. BGP routing optimization reduced inter-region latency by 15ms average. Firewall rules consolidated and optimized, improving packet processing throughput by 30%. Network monitoring enhanced with real-time visibility into traffic flows and performance metrics. Redundant paths established for critical service communications. (Fake message)",
		"Application performance monitoring insights: APM system analyzed 2 million transactions across 50 microservices over 24-hour period. Response time distribution shows 95th percentile under 200ms for critical user journeys. Error rate maintained below 0.1% with automatic retry mechanisms handling transient failures. Memory leak detection prevented 3 potential issues through proactive alerting. Database query optimization suggestions automatically applied, reducing slow query count by 45%. Custom dashboards updated with business-specific KPIs for executive visibility. (Fake message)",
		"Disaster recovery testing and validation: Annual DR exercise successfully completed with full system recovery achieved in 4 hours and 23 minutes, meeting SLA requirements. Database replication verified across geographically distributed backup sites. Application failover mechanisms tested under simulated network partition scenarios. Recovery point objective maintained at 15 minutes with zero data loss. Staff training completed for 25 team members on emergency procedures. Documentation updated with lessons learned and process improvements identified. (Fake message)",
		"DevOps pipeline optimization and automation: CI/CD pipeline enhanced with parallel testing stages, reducing build times from 45 minutes to 18 minutes. Automated security scanning integrated into deployment workflow with policy enforcement gates. Infrastructure as code templates updated to support multi-cloud deployment strategies. Container image vulnerability scanning implemented with automatic base image updates. Deployment rollback mechanisms tested and documented for rapid incident response. Monitoring and alerting configured for pipeline health and performance metrics. (Fake message)",
		"Customer data analytics and insights platform: Real-time analytics engine processed 10 billion events generating actionable insights for 500 business users. Data warehouse performance optimized with columnar storage and intelligent partitioning strategies. Machine learning algorithms identified 15 key customer behavior patterns driving product development decisions. Privacy compliance verified with automated PII detection and masking capabilities. Self-service analytics portal deployed enabling business teams to create custom reports and dashboards independently. (Fake message)",
		"Cloud cost optimization and resource management: Comprehensive cost analysis identified 30% savings opportunity through right-sizing and reserved instance optimization. Automated resource scheduling implemented for non-production environments, reducing off-hours costs by 60%. Multi-cloud strategy deployed across AWS, Azure, and GCP with intelligent workload placement. Tagging policies enforced for cost allocation and chargeback to business units. Unused resource detection and cleanup automation recovered $50,000 monthly cloud spend. (Fake message)",
		"API gateway and microservices architecture evolution: Service mesh deployment completed with Istio providing advanced traffic management and security policies. API versioning strategy implemented supporting backward compatibility for 200 client applications. Rate limiting and throttling policies fine-tuned based on usage patterns and service capacity. Circuit breaker patterns deployed preventing cascade failures during peak traffic events. Service discovery and load balancing optimized for sub-millisecond response times. Observability enhanced with distributed tracing across service boundaries. (Fake message)",
		"Quality assurance and testing automation framework: Test automation coverage increased to 85% with end-to-end scenario validation across web, mobile, and API interfaces. Performance testing infrastructure scaled to simulate 100,000 concurrent users with realistic load patterns. Security testing integrated into CI pipeline with OWASP compliance verification. Cross-browser testing automated across 15 browser and device combinations. Test data management enhanced with synthetic data generation and privacy-safe production data masking. Defect prediction models deployed reducing escaped bugs by 40%. (Fake message)",
		"Enterprise search and knowledge management platform: Elasticsearch cluster scaled to index 50 million documents with sub-second search response times. Natural language processing enhanced search relevance scoring and auto-completion features. Document classification algorithms automatically categorized and tagged 2 million knowledge base articles. Federated search implemented across 10 disparate data sources with unified result ranking. Access control integration ensures users only see authorized content based on role and permissions. Search analytics provide insights into user behavior and content gaps. (Fake message)",
		"Blockchain and distributed ledger implementation: Private blockchain network deployed with 12 validator nodes ensuring transaction immutability and consensus. Smart contracts developed and audited for supply chain transparency and digital asset management. Transaction throughput optimized to handle 10,000 operations per second with finality under 5 seconds. Integration APIs developed for legacy system connectivity with cryptographic proof verification. Governance framework established for network upgrades and protocol changes. Energy-efficient consensus mechanism reduces environmental impact by 75% compared to proof-of-work alternatives. (Fake message)",
		"Internet of Things sensor network deployment: 50,000 IoT sensors deployed across manufacturing facilities providing real-time equipment monitoring and predictive maintenance capabilities. Edge computing infrastructure processes 1 million sensor readings per minute with local analytics and alerting. Time-series database optimized for high-frequency data ingestion and efficient storage compression. Machine learning models identify equipment anomalies 6 hours before failure occurrence. Secure device provisioning and over-the-air update mechanisms ensure fleet management at scale. Digital twin technology provides virtual representation of physical assets for simulation and optimization. (Fake message)",
		"Augmented reality and computer vision platform: AR application framework supports 10,000 concurrent users with real-time object recognition and spatial mapping. Computer vision models trained on 2 million labeled images achieving 98% accuracy for industrial inspection use cases. 3D reconstruction algorithms process point cloud data from depth sensors creating photorealistic virtual environments. Gesture recognition system enables hands-free interaction with 15 distinct command patterns. Cloud-based rendering pipeline delivers high-quality AR experiences across mobile and headset devices. Performance optimization maintains 60fps rendering with sub-20ms motion-to-photon latency. (Fake message)",
	}

	rand.Seed(time.Now().UnixNano())

	switch strings.ToLower(size) {
	case "short":
		return shortMessages[rand.Intn(len(shortMessages))]
	case "medium":
		return mediumMessages[rand.Intn(len(mediumMessages))]
	case "long":
		return longMessages[rand.Intn(len(longMessages))]
	case "random":
		// Randomly choose a size category
		sizes := []string{"short", "medium", "long"}
		randomSize := sizes[rand.Intn(len(sizes))]
		return generateLogMessage(randomSize)
	default:
		return shortMessages[0]
	}
}

// getActualLevel returns the actual level to use, handling "random" option.
func getActualLevel(level string) string {
	if strings.ToLower(level) == "random" {
		levels := []string{"info", "warning", "error"}
		rand.Seed(time.Now().UnixNano())
		return levels[rand.Intn(len(levels))]
	}
	return level
}

// getActualMessage returns the actual message to use, with custom message taking precedence.
func getActualMessage(customMessage, size string) string {
	if customMessage != "" {
		// Custom message takes precedence over size
		if !strings.HasSuffix(customMessage, "(Fake message)") {
			return customMessage + " (Fake message)"
		}
		return customMessage
	}
	return generateLogMessage(size)
}

// generateLogEntry creates a log entry at the specified level.
func generateLogEntry(ctx context.Context, level, message string) {
	// Create logger based on level and decide output stream
	var writer *os.File
	var logLevel zerolog.Level

	switch strings.ToLower(level) {
	case "info":
		writer = os.Stdout
		logLevel = zerolog.InfoLevel
	case "warning":
		writer = os.Stderr
		logLevel = zerolog.WarnLevel
	case "error":
		writer = os.Stderr
		logLevel = zerolog.ErrorLevel
	default:
		writer = os.Stdout
		logLevel = zerolog.InfoLevel
	}

	// Create a logger that writes to the appropriate stream
	logger := zerolog.New(writer).With().Timestamp().Caller().Logger()

	// Log with the appropriate level
	switch logLevel {
	case zerolog.InfoLevel:
		logger.Info().Msg(message)
	case zerolog.WarnLevel:
		logger.Warn().Msg(message)
	case zerolog.ErrorLevel:
		logger.Error().Msg(message)
	}
}
