USE blog;

CREATE TABLE IF NOT EXISTS `comments` (
    id                      BIGINT UNSIGNED     NOT NULL    AUTO_INCREMENT              COMMENT '评论ID',
    article_id              BIGINT UNSIGNED     NOT NULL                                COMMENT '对应的文章ID',
    user_id                 BIGINT UNSIGNED     NOT NULL                                COMMENT '用户ID',
    reply_to_user_id        BIGINT UNSIGNED     NOT NULL    DEFAULT   0                 COMMENT '回复的用户ID',
    content                 TEXT                NOT NULL                                COMMENT '评论内容',
    root_id                 BIGINT UNSIGNED     NOT NULL    DEFAULT   0                 COMMENT '对应的主评论ID',
    ip                      VARCHAR(50)         NOT NULL    DEFAULT ''                  COMMENT '评论IP地址',
    created_time            DATETIME            NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '创建时间',
    updated_time            DATETIME            NOT NULL    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    status                  TINYINT             NOT NULL    DEFAULT   1                 COMMENT '评论状态:0-删除,1-正常',
    PRIMARY KEY (id),
    KEY `idx_articleid_rootid_status` (`article_id`,`root_id`,`status`),
    KEY `idx_rootid_status`(`root_id`,`status`),
    KEY idx_created_time (created_time),
    KEY idx_updated_time (updated_time)
    
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '评论表'