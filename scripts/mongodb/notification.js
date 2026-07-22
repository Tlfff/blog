// 1. 切换到指定的数据库
db = db.getSiblingDB('blog');

// 2. 显式创建通知集合（类似于 CREATE TABLE）
db.createCollection('notifications');

// 3. 创建索引

// 3.1 加速通知列表查询
db.notifications.createIndex(
    { "receiver_id": 1,  "created_time": -1 },//1 表示正序排序，-1表示倒序排序
    { name: "idx_receiverid_createdtime" }
);
// 3.1 加速未读信息修改
db.notifications.createIndex(
    { "receiver_id": 1,  "is_read": 1 },// 排列顺序为 false -> true，因为权重false < true
    { name: "idx_receiverid_isread" }
);


// // 4. 为通知集合创建 TTL（自动过期）索引：30天（2592000秒）后文档自动删除
// db.notifications.createIndex(
//     { "created_time": 1 },
//     { 
//         name: "idx_notification_ttl",
//         expireAfterSeconds: 2592000 
//     }
// );

print(" MongoDB notifications 集合和索引初始化成功 !");