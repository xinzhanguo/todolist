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
	Code    string `json:"code"`
	Style   string `json:"style"`
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
	query := `SELECT id, uid, content, token, tokey, code, style FROM todos WHERE uid = ?`
	row := c.db.QueryRow(query, uid)

	data := &Data{}
	data.UID = uid
	err := row.Scan(&data.ID, &data.UID, &data.Content, &data.Token, &data.Key, &data.Code, &data.Style)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to scan data: %v", err)
	}
	return data, nil
}

func (c *Client) Save(data Data) error {
	query := `INSERT INTO todos (uid, content, token, tokey, code, style) VALUES (?, ?, ?, ?, ?, ?)`
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

func (c *Client) SaveOrUpdate(data Data) error {
	d, err := c.Get(data.UID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Save(data)
		}
		return err
	}
	if d.Token == data.Token {
		return c.Update(data)
	}
	if d.Key == data.Key {
		data.Token = d.Token
		return c.Update(data)
	}
	if d.Code == data.Code {
		data.Token = d.Token
		data.Key = d.Key
		data.Code = d.Code
		return c.Update(data)
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
	if d.Key != "" && d.Key == data.Key {
		d.Token = ""
		return d, nil
	}
	if d.Code != "" && d.Code == data.Code {
		d.ID = 0
		d.Token = ""
		d.Key = ""
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
