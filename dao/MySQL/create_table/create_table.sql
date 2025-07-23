USE logo_api;
CREATE TABLE universities (
    id VARCHAR(20) PRIMARY KEY NOT NULL COMMENT '学校唯一英文简称id',
    title VARCHAR(255) NOT NULL COMMENT '学校中文全称',
    slug CHAR(10) NOT NULL COMMENT '教育部学校识别码',
    vis VARCHAR(255) COMMENT '视觉形象识别系统网址',
    website VARCHAR(255) COMMENT '学校官网网址',
    full_name_en VARCHAR(100) NOT NULL COMMENT '英文官方全称',
    region VARCHAR(10) NOT NULL COMMENT '学校所在大区',
    province VARCHAR(50) NOT NULL COMMENT '学校所在省份',
    city VARCHAR(50) NOT NULL COMMENT '学校所在城市',
    story TEXT COMMENT '学校故事简介',

    has_vector TINYINT(1) DEFAULT 0 COMMENT '是否有矢量格式(svg、ai、eps 等),1=有,0=无',
    main_vector_format VARCHAR(10) COMMENT '主要矢量文件格式，如 svg、ai',
    main_vector_filesize_b INT DEFAULT 0 COMMENT '主要矢量文件大小(B)',
    has_bitmap TINYINT(1) COMMENT '是否有位图格式(png/jpg/webp 等)',
    main_bitmap_format VARCHAR(10) DEFAULT NULL COMMENT '主要位图格式',
    main_bitmap_max_size INT DEFAULT 0 COMMENT '主要位图最大边长(px)',
    main_bitmap_filesize_b INT DEFAULT 0 COMMENT '主要位图文件大小(B)',
    resource_count INT DEFAULT 0 COMMENT '资源文件总数',
    edge_computation_input_id INT DEFAULT NULL COMMENT '边缘计算主输入文件ID(university_resources表)',

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE university_resources (
    id INT PRIMARY KEY AUTO_INCREMENT COMMENT '资源id编号',
    short_name VARCHAR(20) NOT NULL COMMENT '学校唯一英文简称',
    resource_name VARCHAR(50) COMMENT '资源名称',
    resource_type VARCHAR(50) COMMENT '资源类型，如svg、png、zip、rar',
    resource_md5 CHAR(32) COMMENT '资源md5校验',
    resource_size_b INT COMMENT '资源总大小(B)',
    last_update_time TIMESTAMP COMMENT '资源最后更新时间',

    is_vector TINYINT(1) DEFAULT 0 COMMENT '是否为矢量文件',
    is_bitmap TINYINT(1) DEFAULT 0 COMMENT '是否为位图文件',
    resolution VARCHAR(20) DEFAULT NULL COMMENT '分辨率(位图类)',
    used_for_edge TINYINT(1) DEFAULT 0 COMMENT '是否为边缘计算主输入文件',
    is_deleted TINYINT(1) DEFAULT 0 COMMENT '该资源是否已经被删除',
    FOREIGN KEY (short_name) REFERENCES universities(id)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;