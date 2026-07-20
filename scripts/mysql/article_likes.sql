USE blog;

CREATE TABLE IF NOT EXISTS article_likes(
    id                 BIGINT UNSIGNED          NOT NULL    AUTO_INCREMENT              COMMENT 'ID',
    user_id            BIGINT UNSIGNED          NOT NULL                                COMMENT '用户ID',
    article_id         BIGINT UNSIGNED          NOT NULL                                COMMENT '文章ID',
    status             TINYINT                  NOT NULL    DEFAULT 1                   COMMENT '状态:2-未点赞,1-点赞',
    created_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '创建时间',
    updated_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    PRIMARY KEY (id),
    UNIQUE KEY uni_userid_articleid (user_id, article_id),
    
    KEY idx_created_time (created_time),
    KEY idx_updated_time (updated_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '文章点赞表'
