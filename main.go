package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net"
    "time"

    "github.com/armon/go-socks5"
    _ "github.com/go-sql-driver/mysql"
)

type UpstreamConfig struct {
    Host     string
    Port     int
    Username string
    Password string
}

var db *sql.DB

func loadConfig() (*UpstreamConfig, error) {
    config := &UpstreamConfig{}
    err := db.QueryRow(`
        SELECT upstream_host, upstream_port, upstream_user, upstream_password 
        FROM proxy_config 
        ORDER BY updated_at DESC 
        LIMIT 1
    `).Scan(&config.Host, &config.Port, &config.Username, &config.Password)
    return config, err
}

func main() {
    // 初始化数据库连接
    var err error
    db, err = sql.Open("mysql", "proxy_admin:StrongPass123!@tcp(localhost:3306)/proxy_db")
    if err != nil {
        log.Fatal("数据库连接失败:", err)
    }
    defer db.Close()

    // 创建SOCKS5服务器
    server, err := socks5.New(&socks5.Config{
        Credentials: socks5.StaticCredentials{},
        Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
            config, err := loadConfig()
            if err != nil {
                return nil, fmt.Errorf("配置加载失败: %v", err)
            }

            upstreamAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)
            conn, err := net.Dial(network, upstreamAddr)
            if err != nil {
                return nil, err
            }

            // 上游代理认证
            if config.Username != "" && config.Password != "" {
                auth := []byte{0x01, byte(len(config.Username))}
                auth = append(auth, []byte(config.Username)...)
                auth = append(auth, byte(len(config.Password)))
                auth = append(auth, []byte(config.Password)...)
                if _, err := conn.Write(auth); err != nil {
                    conn.Close()
                    return nil, err
                }
            }

            return conn, nil
        },
    })

    if err != nil {
        log.Fatal(err)
    }

    log.Println("代理服务运行在 :9595")
    if err := server.ListenAndServe("tcp", ":9595"); err != nil {
        log.Fatal(err)
    }
}
