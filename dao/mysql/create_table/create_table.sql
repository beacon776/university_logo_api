USE logo_api;
CREATE TABLE IF NOT EXISTS university (
    slug CHAR(10) PRIMARY KEY NOT NULL COMMENT '教育部学校识别码',
    short_name VARCHAR(20) NOT NULL UNIQUE COMMENT '学校唯一英文简称id',
    title VARCHAR(255) NOT NULL UNIQUE COLLATE utf8mb4_zh_0900_as_cs COMMENT '学校中文全称', -- MySQL 8.0 以上
    vis VARCHAR(255) COMMENT '视觉形象识别系统网址',
    website VARCHAR(255) NOT NULL COMMENT '学校官网网址',
    full_name_en VARCHAR(100) NOT NULL COMMENT '英文官方全称',
    region VARCHAR(10) NOT NULL COLLATE utf8mb4_zh_0900_as_cs COMMENT '学校所在大区',
    province VARCHAR(50) NOT NULL COLLATE utf8mb4_zh_0900_as_cs COMMENT '学校所在省份',
    city VARCHAR(50) NOT NULL COLLATE utf8mb4_zh_0900_as_cs COMMENT '学校所在城市',
    story TEXT COLLATE utf8mb4_zh_0900_as_cs COMMENT '学校故事简介',

    has_vector TINYINT DEFAULT 0 COMMENT '是否有矢量格式(svg、ai、eps等),1=有,0=无',
    main_vector_format VARCHAR(10) COMMENT '主要矢量文件格式，如 svg、ai',
    resource_count INT DEFAULT 0 COMMENT '当前学校资源文件总数',
    computation_id INT DEFAULT NULL COMMENT '主计算文件的id(university_resources表)',

    created_time DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_time DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_short_name(short_name),
    INDEX idx_title(title)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS resource (
    id INT PRIMARY KEY AUTO_INCREMENT COMMENT '资源id编号',
    short_name VARCHAR(20) NOT NULL COMMENT '学校唯一英文简称',
    title VARCHAR(255) NOT NULL COLLATE utf8mb4_zh_0900_as_cs COMMENT '学校中文全称',
    name VARCHAR(512) NOT NULL COLLATE utf8mb4_zh_0900_as_cs COMMENT '资源全称（包括.类型）',
    type VARCHAR(50) NOT NULL COMMENT '资源类型，如svg、png、zip、rar',
    md5 CHAR(32) NOT NULL COMMENT '资源md5校验值',
    size INT NOT NULL COMMENT '资源大小(单位为B)',
    last_update_time TIMESTAMP NOT NULL COMMENT '资源最后更新时间',
    is_vector TINYINT NOT NULL DEFAULT 0 COMMENT '是否为矢量文件',
    is_bitmap TINYINT NOT NULL DEFAULT 0 COMMENT '是否为位图文件',
    width INT DEFAULT 0 COMMENT '宽度(px)',
    height INT DEFAULT 0 COMMENT '高度(px)',
    used_for_edge TINYINT DEFAULT 0 COMMENT '是否为边缘计算主输入文件',
    is_deleted TINYINT NOT NULL DEFAULT 0 COMMENT '该资源是否已经被删除',
    background_color VARCHAR(20) NOT NULL COMMENT '背景颜色，可为十六进制或CSS颜色名',
    FOREIGN KEY (short_name) REFERENCES university(short_name),
    FOREIGN KEY (title) REFERENCES university(title)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user (
    id INT PRIMARY KEY AUTO_INCREMENT COMMENT '用户id',
    status INT COMMENT '用户启用状态 1 启用 0 禁用',
    username VARCHAR(50) NOT NULL COMMENT '用户名',
    password VARCHAR(256) NOT NULL COMMENT '用户密码'
)