-- 创建casbin规则表（PostgreSQL版本）
CREATE TABLE IF NOT EXISTS casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(255) NOT NULL DEFAULT '',
    v0 VARCHAR(255) NOT NULL DEFAULT '', -- 角色ID/用户ID
    v1 VARCHAR(255) NOT NULL DEFAULT '', -- 资源路径
    v2 VARCHAR(255) NOT NULL DEFAULT '', -- 访问方法
    v3 VARCHAR(255) NOT NULL DEFAULT '',
    v4 VARCHAR(255) NOT NULL DEFAULT '',
    v5 VARCHAR(255) NOT NULL DEFAULT ''
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_casbin_ptype ON casbin_rule (ptype);
CREATE INDEX IF NOT EXISTS idx_casbin_v0 ON casbin_rule (v0);
CREATE INDEX IF NOT EXISTS idx_casbin_v1 ON casbin_rule (v1);

-- 初始化RBAC策略
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
('p', 'admin', '/*', '*'),
('p', 'user', '/jobs', 'GET'),
('p', 'user', '/jobs/*', 'GET'),
('g', 'alice', 'admin', ''),
('g', 'bob', 'user', '');

-- 创建角色表（PostgreSQL版本）
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255)
);

-- 创建用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    user_id INT NOT NULL,
    role_id INT NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- 初始化基础角色
INSERT INTO roles (name, description) VALUES
('admin', '系统管理员'),
('user', '普通用户'); 