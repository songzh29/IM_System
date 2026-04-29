package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// 入库计数(成功/失败 用 label 区分)
	MessagesPersistedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_messages_persisted_total",
			Help: "Total number of messages persisted to DB",
		},
		[]string{"result"}, // success / failure
	)

	// 投递计数(本地/跨实例/离线 用 label 区分)
	MessagesDeliveredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_messages_delivered_total",
			Help: "Total number of messages delivered, by path",
		},
		[]string{"path"}, // local / cross_instance / offline
	)

	// 投递失败计数(按原因分类)
	MessagesFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "im_messages_failed_total",
			Help: "Total number of message delivery failures",
		},
		[]string{"reason"}, // chan_full / forward_publish_failed / write_failed / ...
	)

	// 在线用户数(无 label)
	OnlineUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "im_online_users",
			Help: "Current number of online users",
		})

	// PublishForward 成功后 Inc
	ForwardPublishedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "im_redis_pub",
			Help: "total numbers of Redis published",
		})

	// handlerForwardMsg 入口 Inc(无论本节点是否目标)
	ForwardReceivedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "im_redis_sub",
			Help: "total numbers of every Redis node received",
		})

	// 定时(每 30 秒)从 Redis 查 KEYS online:user:* 的数量
	OnlineUsersRedis = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "im_online_users_in_redis",
			Help: "Current number of online users in Redis",
		})

	//查看send channel中的数量
	SendChanUsage = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "im_send_channel_histogram",
			Help:    "Histogram of client send channel",
			Buckets: []float64{10, 50, 100, 150, 200, 250, 256},
		})
)

func Init() {
	prometheus.MustRegister(MessagesPersistedTotal)
	prometheus.MustRegister(MessagesDeliveredTotal)
	prometheus.MustRegister(MessagesFailedTotal)
	prometheus.MustRegister(OnlineUsers)
	prometheus.MustRegister(ForwardPublishedTotal)
	prometheus.MustRegister(ForwardReceivedTotal)
	prometheus.MustRegister(OnlineUsersRedis)
	prometheus.MustRegister(SendChanUsage)

	// 预初始化 label,即使没事件,/metrics 也会显示为 0
	MessagesPersistedTotal.WithLabelValues("success").Add(0)
	MessagesPersistedTotal.WithLabelValues("failure").Add(0)
	MessagesDeliveredTotal.WithLabelValues("local").Add(0)
	MessagesDeliveredTotal.WithLabelValues("cross_instance").Add(0)
	MessagesDeliveredTotal.WithLabelValues("offline").Add(0)
}
