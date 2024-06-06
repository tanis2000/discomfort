package service

import (
    "encoding/json"
    "os"
)

type ImageDB struct {
    Images []string `json:"images"`
}

func NewImageDB() *ImageDB {
    return &ImageDB{}
}

func (i *ImageDB) Add(image string) {
    i.Images = append(i.Images, image)
}

func (i *ImageDB) Save() error {
    b, err := json.Marshal(i)
    if err != nil {
        return err
    }

    err = os.WriteFile("images.json", b, 0644)
    if err != nil {
        return err
    }

    return nil
}

func (i *ImageDB) Load() error {
    b, err := os.ReadFile("images.json")
    if err != nil {
        return err
    }
    err = json.Unmarshal(b, i)
    return nil
}
