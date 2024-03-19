package main

import (
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net/http"
	"time"
)

var DB *gorm.DB
var IdWorker *snowflake.Node

func main() {
	r := gin.New()
	r.GET("/student", GetAllStudent)
	r.POST("/student", AddStudent)
	r.POST("/checkin", Check)
	r.GET("/tally", Tally)
	r.Run()
}

type Student struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (Student) TableName() string {
	return "student"
}

type TodayStudent struct {
	Student
	State int8 `json:"state"`
}

// GetAllStudent 全部学生列表
func GetAllStudent(c *gin.Context) {
	var students []*TodayStudent
	DB.Table("student").Scan(&students)
	for _, student := range students {
		var c Checkin
		r := DB.Table("checkin").Where("student_id = ? AND checkin_date = ?", student.ID, time.Now().Format(time.DateOnly)).Scan(&c)
		if r.RowsAffected > 0 {
			student.State = c.State
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "result": students})
}

// AddStudent 添加学生信息
func AddStudent(c *gin.Context) {
	var s Student
	s.ID = IdWorker.Generate().String()
	c.ShouldBindJSON(&s)
	var student Student
	r := DB.Table("student").Where("name = ?", s.Name).First(&student)
	if r.RowsAffected > 0 {
		c.JSON(http.StatusOK, gin.H{"code": http.StatusAlreadyReported, "message": "数据已存在"})
		return
	}
	DB.Create(&s)
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "result": s})
}

type Checkin struct {
	ID          string    `json:"id"`
	StudentId   string    `json:"studentId"`
	CheckinDate time.Time `json:"checkinDate"`
	State       int8      `json:"state"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (Checkin) TableName() string {
	return "checkin"
}

// Check 标记打卡
func Check(c *gin.Context) {
	var checkIn Checkin
	checkIn.ID = IdWorker.Generate().String()
	c.ShouldBindJSON(&checkIn)
	fmt.Println(checkIn)
	var ck Checkin
	r := DB.Table("checkin").Where("student_id = ? AND checkin_date = ?", checkIn.StudentId, checkIn.CheckinDate.Format(time.DateOnly)).Scan(&ck)
	if r.RowsAffected > 0 {
		c.JSON(http.StatusOK, gin.H{"code": http.StatusNotModified, "message": "今日数据已存在"})
		return
	}
	DB.Create(&checkIn)
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "result": checkIn})
}

type TallyData struct {
	Student
	Span []*Data `json:"span" gorm:"-"` // key:日期,value:打卡状态
}

type Data struct {
	CheckinDate time.Time `json:"checkinDate"`
	Date        string    `json:"date"`
	State       int8      `json:"state"`
}

// Tally 按照时间维度统计每个学生的打卡情况
func Tally(c *gin.Context) {
	beginDate := c.Query("beginDate")
	endDate := c.Query("endDate")
	var tallyDataList []*TallyData
	DB.Table("student").Scan(&tallyDataList)
	for _, s := range tallyDataList {
		var data []*Data
		r := DB.Table("checkin").Select("checkin_date, state").Where("student_id = ? AND checkin_date >= ? AND checkin_date <= ?", s.ID, beginDate, endDate).Scan(&data)
		if r.RowsAffected > 0 {
			for _, d := range data {
				d.Date = d.CheckinDate.Format(time.DateOnly)
			}
			s.Span = data

		} else {
			s.Span = nil
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "result": tallyDataList})
}

func init() {
	db, err := gorm.Open(mysql.Open("root:rootroot@tcp(127.0.0.1:3306)/student_checkin?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	DB = db
	node, _ := snowflake.NewNode(1)
	IdWorker = node
}
