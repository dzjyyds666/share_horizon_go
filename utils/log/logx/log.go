package logx

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CustomLogger struct {
	MaxSize    int    `json:"max_size"`    // 最大日志文件大小
	MaxAge     int    `json:"max_age"`     // 保留几天
	MaxBackups int    `json:"max_backups"` // 保留多少个文件
	Filename   string `json:"filename"`    // 日志文件名
}

func GetLogger(filename string) *CustomLogger {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("获取当前用户目录失败:%v", err)
	}

	filename = filepath.Join(homeDir, "logs", filename+".log")

	return &CustomLogger{
		MaxSize:    10,
		MaxAge:     10,
		MaxBackups: 10,
		Filename:   filename,
	}
}

// 写入info日志
func (cl *CustomLogger) Infof(format string, message ...interface{}) {

	level := "[INFO]"

	msg := fmt.Sprintf(format, message...)

	// 写入文件
	go WriteToFile(cl.Filename, msg, level, cl)

	log.SetOutput(os.Stdout)
	log.SetPrefix(level)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Println(msg)
}

func (cl *CustomLogger) Info(message interface{}) {
	msg := fmt.Sprintf("%v", message)
	cl.Infof("%v", msg)
}

// 写入error日志
func (cl *CustomLogger) Errorf(format string, message ...interface{}) {
	msg := fmt.Sprintf(format, message...)

	level := "[ERROR]"
	// 写入文件
	go WriteToFile(cl.Filename, msg, level, cl)

	log.SetOutput(os.Stdout)
	log.SetPrefix(level)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Println(msg)
}

func (cl *CustomLogger) Error(message interface{}) {
	msg := fmt.Sprintf("%v", message)
	cl.Errorf("%v", msg)
}

// 写入debug日志
func (cl *CustomLogger) Debugf(format string, message ...interface{}) {
	msg := fmt.Sprintf(format, message...)

	level := "[DEBUG]"
	// 写入文件
	go WriteToFile(cl.Filename, msg, level, cl)

	log.SetOutput(os.Stdout)
	log.SetPrefix(level)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Println(msg)
}

func (cl *CustomLogger) Debug(message interface{}) {
	msg := fmt.Sprintf("%v", message)
	cl.Debugf("%v", msg)
}

// 写入warn日志
func (cl *CustomLogger) Warnf(format string, message ...interface{}) {
	msg := fmt.Sprintf(format, message...)

	level := "[WARN]"
	// 写入文件
	go WriteToFile(cl.Filename, msg, level, cl)

	log.SetOutput(os.Stdout)
	log.SetPrefix(level)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Println(msg)
}

func (cl *CustomLogger) Warn(message interface{}) {
	msg := fmt.Sprintf("%v", message)
	cl.Warnf("%v", msg)
}

// 写入fatal日志
func (cl *CustomLogger) Fatalf(format string, message ...interface{}) {
	msg := fmt.Sprintf(format, message...)

	level := "[FATAL]"
	// 写入文件
	go WriteToFile(cl.Filename, msg, level, cl)

	log.SetOutput(os.Stdout)
	log.SetPrefix(level)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Fatalln(msg)
}

func (cl *CustomLogger) Fatal(message interface{}) {
	msg := fmt.Sprintf("%v", message)
	cl.Fatalf("%v", msg)
}

// 日志写入文件
func WriteToFile(logPath string, logMsg string, level string, cl *CustomLogger) {

	// 检查日志文件
	cl.checkLogFile()

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Printf("打开文件失败:%v", err)
		return
	}
	defer file.Close()

	//设置日志输出
	log.SetPrefix(level)
	log.SetFlags(log.Ldate | log.Ltime)
	log.SetOutput(file)

	log.Println(logMsg)
}

func (cl *CustomLogger) checkLogFile() {
	//确保目录存在
	dir := filepath.Dir(cl.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建目录失败:%v", err)
	}

	stat, err := os.Lstat(cl.Filename)
	if os.IsNotExist(err) {
		file, err := os.Create(cl.Filename)
		if err != nil {
			fmt.Printf("创建文件失败:%v", err)
		}
		defer file.Close()
	} else if stat.Size() > int64(1024*1024*cl.MaxSize) {

		// 获取当前备份文件列表
		pattern := cl.Filename + ".*"
		backups, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Printf("获取备份文件列表失败: %v", err)
		}

		// 删除多余的备份文件
		if len(backups) >= cl.MaxBackups {
			sort.Strings(backups)
			for i := 0; i <= len(backups)-cl.MaxBackups; i++ {
				if err := os.Remove(backups[i]); err != nil {
					fmt.Printf("删除备份文件失败: %v", err)
				}
			}
		}

		// 备份旧文件
		names := strings.Split(cl.Filename, ".")
		backupName := names[0] + "_" + time.Now().Format("20060102150405") + "." + names[1]
		err = os.Rename(cl.Filename, backupName)
		if err != nil {
			fmt.Printf("Logx Error:%v", err)
			return
		}

		// 创建新文件
		file, err := os.Create(cl.Filename)
		if err != nil {
			fmt.Printf("Logx Error:%v", err)
			return
		}
		defer file.Close()
	}
}
