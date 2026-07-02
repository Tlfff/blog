CREATE TABLE users(
    id              BIGINT UNSIGNED     NOT NULL    AUTO_INCREMENT              COMMENT '用户ID',
    nickname        VARCHAR(50)         NOT NULL                                COMMENT '用户名',
    phone           VARCHAR(50)         NOT NULL                                COMMENT '手机号',
    password        VARCHAR(255)        NOT NULL                                COMMENT '加密后密码:算法$迭代次数$Salt$Hash',
    avatar          VARCHAR(255)                                                COMMENT '头像URL地址',
    role            TINYINT             NOT NULL    DEFAULT 1                   COMMENT '用户角色:1-普通用户,2-管理员',
    created_time    DATETIME            NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '创建时间',
    updated_time    DATETIME            NOT NULL    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    status          TINYINT             NOT NULL                                COMMENT '用户状态:0-删除,1-正常',
    last_login_ip   VARCHAR(50)         NOT NULL    DEFAULT ''                  COMMENT '最后登录IP',
    last_login_time DATETIME            NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '最后登录时间',
    PRIMARY KEY (id),
    -- 唯一索引：电话号码
    UNIQUE KEY uni_phone (phone),
    -- 唯一索引：昵称
    UNIQUE KEY uni_nickname (nickname),
    KEY idx_created_time (created_time),
    KEY idx_updated_time (updated_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '用户表'
