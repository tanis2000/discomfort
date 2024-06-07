package database

import "github.com/google/uuid"

type Image struct {
    ID           string
    UserID       int
    IsPublic     bool
    Filename     string
    FriendlyName string
}

func NewImage(u *User, filename string, friendlyName string, isPublic bool) *Image {
    res := &Image{
        ID:           uuid.New().String(),
        UserID:       u.ID,
        IsPublic:     isPublic,
        Filename:     filename,
        FriendlyName: friendlyName,
    }
    return res
}
