package db

import (
	"database/sql"
	"fmt"
)

type Client struct {
	db *sql.DB
}

type Data struct {
	ID      int64  `json:"id"`
	UID     string `json:"uid"`
	Content string `json:"content"`
	Token   string `json:"token"`
	Key     string `json:"key"`
	Code    string `json:"sharecode"`
	Style   string `json:"style"`
	Version int64  `json:"version"`
}

type Chat struct {
	ID      int64  `json:"id"`
	UID     string `json:"uid"` // 谁收
	Content string `json:"content"`
	Token   string `json:"token"` //
	Creator string `json:"creator"`
}

const (
	NEEDKEY  = "not allowed,need key"
	NEEDCODE = "not allowed,need code"
)

func New(dsn string) (*Client, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &Client{db: db}, nil
}

func (c *Client) Get(uid string) (*Data, error) {
	query := `SELECT id, uid, content, version, token, tokey, code, style FROM todos WHERE uid = ?`
	row := c.db.QueryRow(query, uid)

	data := &Data{}
	data.UID = uid
	err := row.Scan(&data.ID, &data.UID, &data.Content, &data.Version, &data.Token, &data.Key, &data.Code, &data.Style)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to scan data: %v", err)
	}
	return data, nil
}

func (c *Client) Save(data Data) error {
	query := `INSERT INTO todos (uid, content, token, tokey, code, style, version) VALUES (?, ?, ?, ?, ?, ?, 1)`
	result, err := c.db.Exec(query, data.UID, data.Content, data.Token, data.Key, data.Code, data.Style)
	if err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	data.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	fmt.Printf("Data saved with ID: %d\n", data.ID)
	return nil
}

func (c *Client) Update(data Data) error {
	query := `UPDATE todos SET content = ?, token = ?, tokey = ?, code = ? WHERE uid = ?`
	result, err := c.db.Exec(
		query,
		data.Content,
		data.Token,
		data.Key,
		data.Code,
		data.UID,
	)
	if err != nil {
		return fmt.Errorf("failed to update data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected, data with ID %s not found", data.UID)
	}

	fmt.Printf("Data updated successfully, rows affected: %d\n", rowsAffected)
	return nil
}

func (c *Client) UpdateContent(data Data) error {
	query := `UPDATE todos SET content = ?, version = version+1 WHERE uid = ? and version = ?`
	result, err := c.db.Exec(
		query,
		data.Content,
		data.UID,
		data.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected, data with ID %s not found", data.UID)
	}

	fmt.Printf("Data updated successfully, rows affected: %d\n", rowsAffected)
	return nil
}

func (c *Client) SaveOrUpdate(data Data) error {
	d, err := c.Get(data.UID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Save(data)
		}
		return err
	}
	if d.Version != data.Version {
		return fmt.Errorf("version is diff")
	}
	if d.Token == data.Token || d.Key == data.Key || d.Code == data.Code || d.Key == "" {
		return c.UpdateContent(data)
	}
	return fmt.Errorf("not allowed")
}

func (c *Client) GetAllowed(data Data) (*Data, error) {
	d, err := c.Get(data.UID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &Data{UID: data.UID}, nil
		}
		return nil, err
	}
	d.ID = 0
	if d.Token == data.Token {
		return d, nil
	}
	d.Token = ""
	if d.Key != "" && d.Key == data.Key {
		d.Key = ""
		return d, nil
	}
	if d.Code != "" && d.Code == data.Code {
		d.Key = ""
		d.Code = ""
		return d, nil
	}
	if d.Code != "" {
		return nil, fmt.Errorf(NEEDCODE)
	}
	if d.Key != "" {
		return nil, fmt.Errorf(NEEDKEY)
	}
	d.Token = ""
	d.Key = ""
	d.Code = ""
	return d, nil
}

func (c *Client) Set(uid, key, val string) error {
	query := fmt.Sprintf("UPDATE todos SET  %s = ? WHERE uid = ?", key)
	result, err := c.db.Exec(
		query,
		val,
		uid,
	)
	if err != nil {
		return fmt.Errorf("failed to update data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected, data with ID %s not found", uid)
	}

	fmt.Printf("Data updated successfully, rows affected: %d\n", rowsAffected)
	return nil
}

func (c *Client) SetKey(data Data, key string) error {
	d, err := c.Get(data.UID)
	if err != nil {
		return err
	}
	if d.Token == data.Token {
		return c.Set(data.UID, "tokey", key)
	}
	if d.Key != "" || d.Key == data.Key {
		return c.Set(data.UID, "tokey", key)
	}
	return fmt.Errorf("not allowed")
}

func (c *Client) SetCode(data Data, code string) error {
	d, err := c.Get(data.UID)
	if err != nil {
		return err
	}
	if d.Token == data.Token {
		return c.Set(data.UID, "code", code)
	}
	if d.Key != "" || d.Key == data.Key {
		return c.Set(data.UID, "code", code)
	}
	return fmt.Errorf("not allowed")
}

func (c *Client) SetStyle(data Data, style string) error {
	d, err := c.Get(data.UID)
	if err != nil {
		return err
	}
	if d.Token == data.Token {
		return c.Set(data.UID, "style", style)
	}
	if d.Key != "" || d.Key == data.Key {
		return c.Set(data.UID, "style", style)
	}
	if d.Code != "" || d.Code == data.Code {
		return c.Set(data.UID, "style", style)
	}
	return fmt.Errorf("not allowed")
}

func (c *Client) GetVersion(data Data) (int64, error) {
	d, err := c.Get(data.UID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if d.Token == data.Token || d.Key == data.Key || d.Code == data.Code {
		return d.Version, nil
	}
	return 0, fmt.Errorf("not allowed")
}

func (c *Client) SendChat(chat Chat) error {
	query := `INSERT INTO chats (uid, content, token, creator) VALUES (?, ?, ?, ?)`
	result, err := c.db.Exec(query, chat.UID, chat.Content, chat.Token, chat.Creator)
	if err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	chat.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	fmt.Printf("Data saved with ID: %d\n", chat.ID)
	return nil
}

func (c *Client) GetChat(uid, token string) ([]Chat, error) {
	query := `SELECT id,content,creator FROM chats WHERE uid = ? AND token=? ORDER BY id DESC limit 100`
	rows, err := c.db.Query(query, uid, token)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 遍历查询结果
	var chats []Chat
	for rows.Next() {
		var chat Chat
		err := rows.Scan(&chat.ID, &chat.Content, &chat.Creator)
		if err != nil {
			return nil, err
		}
		if chat.Creator != token {
			chat.Creator = "other"
		} else {
			chat.Creator = "user"
		}
		chats = append(chats, chat)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}
