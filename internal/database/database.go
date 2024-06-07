package database

import (
    "context"
    "database/sql"
    "discomfort/internal/imagedb"
)
import _ "github.com/chaisql/chai/driver"
import _ "github.com/mattn/go-sqlite3"

type Database struct {
    db *sql.DB
}

func NewDatabase() *Database {
    return &Database{}
}

func (d *Database) Open() error {
    var err error
    d.db, err = sql.Open("sqlite3", "data/discomfort.sqlite")
    if err != nil {
        return err
    }
    return nil
}

func (d *Database) Close() error {
    if d.db != nil {
        err := d.db.Close()
        if err != nil {
            return err
        }
    }
    return nil
}

func (d *Database) Setup() error {
    _, err := d.db.ExecContext(context.TODO(), `
    CREATE TABLE IF NOT EXISTS settings (
        id int PRIMARY KEY,
        version int,
        migrated_from_imagedb bool default false
    );
    `)
    if err != nil {
        return err
    }

    _, err = d.db.ExecContext(context.TODO(), `
    CREATE TABLE IF NOT EXISTS users (
        id int PRIMARY KEY
    );
    `)
    if err != nil {
        return err
    }

    _, err = d.db.ExecContext(context.TODO(), `
    CREATE TABLE IF NOT EXISTS images (
        id TEXT PRIMARY KEY,
        user_id int,
        is_public bool DEFAULT false,
        filename TEXT DEFAULT '',
        friendly_name TEXT DEFAULT '',
        CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id)
    );
    `)
    if err != nil {
        return err
    }

    err = d.setupSettings()
    if err != nil {
        return err
    }

    err = d.migrateFromImageDB()
    if err != nil {
        return err
    }
    return nil
}

func (d *Database) FindUerById(id int) (User, error) {
    res := d.db.QueryRow("SELECT * FROM users WHERE id = ?", id)
    var u User
    if err := res.Scan(&u.ID); err != nil {
        return u, err
    }
    return u, nil
}

func (d *Database) AddUser(u *User) (int64, error) {
    res, err := d.db.Exec("INSERT INTO users (id) VALUES (?)", u.ID)
    if err != nil {
        return 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, err
    }
    return id, nil
}

func (d *Database) AddImage(i *Image) (int64, error) {
    res, err := d.db.Exec("INSERT INTO images (id, user_id, is_public, filename, friendly_name) VALUES (?,?,?,?,?)", i.ID, i.UserID, i.IsPublic, i.Filename, i.FriendlyName)
    if err != nil {
        return 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, err
    }
    return id, nil
}

func (d *Database) FindImageById(id int) (Image, error) {
    res := d.db.QueryRow("SELECT id, user_id, filename, friendly_name FROM images WHERE id = ?", id)
    var i Image
    if err := res.Scan(&i.ID, &i.UserID, &i.Filename, &i.FriendlyName); err != nil {
        return i, err
    }
    return i, nil
}

func (d *Database) FindImagesBySearchTerm(term string) ([]Image, error) {
    res := make([]Image, 0)
    searchTerm := "%" + term + "%"
    rows, err := d.db.Query("SELECT * FROM images WHERE filename LIKE ? OR friendly_name LIKE ?", searchTerm, searchTerm)
    if err != nil {
        return res, err
    }
    defer rows.Close()
    for rows.Next() {
        var i Image
        if err := rows.Scan(&i); err != nil {
            return res, err
        }
        res = append(res, i)
    }
    return res, nil
}

func (d *Database) getSettings() (*Setting, error) {
    res := d.db.QueryRow("SELECT id, version, migrated_from_imagedb FROM settings LIMIT 1")
    var s Setting
    if err := res.Scan(&s.ID, &s.Version, &s.MigratedFromImageDB); err != nil {
        return nil, err
    }
    return &s, nil
}

func (d *Database) addSettings(s *Setting) (int64, error) {
    res, err := d.db.Exec("INSERT INTO settings (id, version, migrated_from_imagedb) VALUES (?,?,?)", s.ID, s.Version, s.MigratedFromImageDB)
    if err != nil {
        return 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, err
    }
    return id, nil
}

func (d *Database) updateSettings(s *Setting) (int64, error) {
    res, err := d.db.Exec("UPDATE settings SET version = ?, migrated_from_imagedb=? WHERE id=?", s.Version, s.MigratedFromImageDB, s.ID)
    if err != nil {
        return 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, err
    }
    return id, nil
}

func (d *Database) setupSettings() error {
    s, err := d.getSettings()
    if err != nil {
        s = &Setting{
            ID:                  1,
            Version:             1,
            MigratedFromImageDB: false,
        }
        _, err = d.addSettings(s)
        if err != nil {
            return err
        }
    }

    _, err = d.FindUerById(1)
    if err != nil {
        // Create a service user to use for orphan images and stuff like that
        u := &User{ID: 1}
        _, err = d.AddUser(u)
        if err != nil {
            return err
        }
    }
    return nil
}

func (d *Database) migrateFromImageDB() error {
    s, err := d.getSettings()
    if err != nil {
        return err
    }
    if s.MigratedFromImageDB {
        return nil
    }
    u, err := d.FindUerById(1)
    if err != nil {
        return err
    }
    imageDB := imagedb.NewImageDB()
    err = imageDB.Load()
    if err != nil {
        return err
    }
    for _, image := range imageDB.Images {
        i := NewImage(&u, image, image, true)
        _, err = d.AddImage(i)
        if err != nil {
            return err
        }
    }
    s.MigratedFromImageDB = true
    _, err = d.updateSettings(s)
    if err != nil {
        return err
    }
    return nil
}
