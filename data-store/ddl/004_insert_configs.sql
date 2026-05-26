-- 预置配置数据
-- 注意：encrypted=1 的配置需要通过 POST /config/encrypted 接口写入（会自动加密）
-- 以下 encrypted=0 的明文配置可以直接 INSERT

-- auth-server 配置
INSERT INTO configs (service, `key`, value, encrypted) VALUES
('auth-server', 'grpc', '{"port":9090,"cert":"../certs/server.pem","cert_key":"../certs/server-key.pem"}', 0),
('auth-server', 'jwt', '{"secret":"my-secret-key-for-ttuser-2024","access_expire":"2h","refresh_expire":"168h"}', 0)
ON DUPLICATE KEY UPDATE value = VALUES(value), version = version + 1;

-- event-producer 配置
INSERT INTO configs (service, `key`, value, encrypted) VALUES
('event-producer', 'rocketmq', '{"name_server":"127.0.0.1:9876","group_name":"ttuser-producer-group"}', 0)
ON DUPLICATE KEY UPDATE value = VALUES(value), version = version + 1;

-- async-handler 配置
INSERT INTO configs (service, `key`, value, encrypted) VALUES
('async-handler', 'rocketmq', '{"name_server":"127.0.0.1:9876","consumer_group":"sms-consumer-group"}', 0),
('async-handler', 'sms', '{"api_key":"your-api-key","api_secret":"your-api-secret","sign_name":"TT用户平台","template":"尊敬的%s，您已成功注册TT用户平台，欢迎使用！"}', 0)
ON DUPLICATE KEY UPDATE value = VALUES(value), version = version + 1;

-- proc 配置
INSERT INTO configs (service, `key`, value, encrypted) VALUES
('proc', 'server', '{"port":8080}', 0),
('proc', 'auth-client', '{"addr":"localhost:9090","ca_cert":"../certs/ca.pem"}', 0)
ON DUPLICATE KEY UPDATE value = VALUES(value), version = version + 1;

-- 敏感配置（需要通过加密接口写入，以下仅示例明文值，实际应调用 POST /config/encrypted）
-- auth-server mysql: {"dsn":"root:123456@tcp(localhost:3306)/ttuser?charset=utf8mb4&parseTime=true&loc=Local"}
-- async-handler sms api_key/secret: 通过加密接口单独设置
