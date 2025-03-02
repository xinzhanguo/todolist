package db

import (
	"database/sql"
)

type Client struct {
	db *sql.DB
}

func New(dsn string) (*Client, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &Client{db: db}, nil
}

func (c *Client) Save(uid string, content string) error {
	var id int64
	if err := c.db.QueryRow("SELECT id FROM todos WHERE uid = ?", uid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			// 数据不存在，执行插入
			_, err := c.db.Exec("INSERT INTO todos (uid, content) VALUES (?, ?)", uid, content)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	// 数据存在，执行更新
	if _, err := c.db.Exec("UPDATE todos SET content = ? WHERE uid = ?", content, uid); err != nil {
		return err
	}
	return nil

}

func (c *Client) Query(uid string) (string, error) {
	var content string
	query := "SELECT content FROM todos WHERE uid = ?"
	if err := c.db.QueryRow(query, uid).Scan(&content); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return content, nil
}
