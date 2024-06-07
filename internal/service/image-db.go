package service

import (
    "encoding/json"
    "errors"
    "os"
)

type ImageDB struct {
    Images []string `json:"images"`
}

func NewImageDB() *ImageDB {
    return &ImageDB{
        Images: make([]string, 0),
    }
}

func (i *ImageDB) Add(image string) {
    i.Images = append(i.Images, image)
}

func (i *ImageDB) Save() error {
    b, err := json.Marshal(i)
    if err != nil {
        return err
    }

    err = os.WriteFile("data/images.json", b, 0644)
    if err != nil {
        return err
    }

    return nil
}

func (i *ImageDB) Load() error {
    b, err := os.ReadFile("data/images.json")
    if errors.Is(err, os.ErrNotExist) {
        return nil
    }
    if err != nil {
        return err
    }
    err = json.Unmarshal(b, i)
    if err != nil {
        return err
    }
    return nil
}
