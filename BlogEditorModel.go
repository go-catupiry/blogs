package blog

import (
	"time"
)

/******sql******
CREATE TABLE `blogs-editors` (
  `createdAt` datetime NOT NULL,
  `updatedAt` datetime NOT NULL,
  `userId` int(11) NOT NULL,
  `blogId` int(11) NOT NULL,
  PRIMARY KEY (`userId`,`blogId`),
  KEY `blogId` (`blogId`),
  CONSTRAINT `blogs-editors_ibfk_1` FOREIGN KEY (`userId`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `blogs-editors_ibfk_2` FOREIGN KEY (`blogId`) REFERENCES `blogs` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='We.js users table'
******sql******/
// BlogEditorModel We.js users table
type BlogEditorModel struct {
	CreatedAt time.Time `gorm:"column:createdAt;type:datetime;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt;type:datetime;not null" json:"updatedAt"`
	UserID    int       `gorm:"primaryKey;column:userId;type:int(11);not null" json:"userId"`
	// Users     Users     `gorm:"joinForeignKey:userId;foreignKey:id" json:"usersList"` // We.js users table
	BlogID int `gorm:"primaryKey;index:blogId;column:blogId;type:int(11);not null" json:"blogId"`
	// Blogs     Blogs     `gorm:"joinForeignKey:blogId;foreignKey:id" json:"blogsList"`
}

// TableName get sql table name
func (m *BlogEditorModel) TableName() string {
	return "blogs-editors"
}
