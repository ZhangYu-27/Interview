package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	time2 "time"
)

type logInfo struct {
	nowTime  time2.Time
	threadId int
}

var logAddr = "/tmp/test/"
var timeArr = [5]time2.Duration{1990, 1980, 2010, 2020, 2000}
var sig = make(chan os.Signal)
var workOver = make(chan os.Signal)

func main() {
	wg := &sync.WaitGroup{}
	mt := &sync.Mutex{}
	var threads int
	fmt.Println("请输入要创建的线程数量，回车确认")
	_, err := fmt.Scanln(&threads)
	if threads == 0 || err != nil {
		fmt.Println("输入错误，程序退出")
		return
	}
	signal.Notify(sig, syscall.SIGINT, syscall.SIGUSR1)
	go func() {
		isOver := false
		for {
			if isOver {
				break
			}
			select {
			case s := <-sig:
				for i := 0; i < threads*2; i++ {
					workOver <- s
				}
				isOver = true
			}
		}
	}()
	for i := 0; i < threads; i++ {
		fmt.Println("创建协程", i)
		wg.Add(1)
		var workCommand = make(chan logInfo)
		go withKeyboardInputWriteLog(wg, i, mt, workCommand)
		go timeSleep(workCommand, i)
	}
	wg.Wait()
	fmt.Println("退出程序")
}

func timeSleep(workCommand chan logInfo, i int) {
	var isBreak bool
	for {
		if isBreak {
			break
		}
		select {
		case _ = <-workCommand:
			close(workCommand)
			isBreak = true
			break
		default:
			//休眠两秒左右的时间
			key := rand.Intn(4)
			timeMs := timeArr[key]
			time2.Sleep(time2.Millisecond * timeMs)
			//组建包含log的结构体
			sendInfo := logInfo{
				nowTime:  time2.Now(),
				threadId: i,
			}
			//更新lastTime
			workCommand <- sendInfo
		}
	}
}

func writeLog(logInfo []byte, mt *sync.Mutex) error {
	mt.Lock()
	defer mt.Unlock()

	fl, err := os.OpenFile(logAddr+"log.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if os.IsNotExist(err) {
		err = os.Mkdir(logAddr, 0777)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = fl.Write(logInfo)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func withKeyboardInputWriteLog(wg *sync.WaitGroup, threadId int, mt *sync.Mutex, workCommand chan logInfo) {
	defer wg.Done()
	var writerNum int
	lastTime := time2.Now()
	var isBreak bool
	for {
		select {
		case _ = <-workOver:
			fmt.Println("退出协程", threadId)
			intervalTime := time2.Now().UnixMilli() - lastTime.UnixMilli()
			intervalTimeFloat := float64(intervalTime)
			lastSeconds := intervalTimeFloat / 1000
			str := fmt.Sprintf(time2.Now().String() + " 线程id" + strconv.Itoa(threadId) + " 距离上次此协程发送" +
				strconv.FormatFloat(lastSeconds, 'f', -1, 64) + "秒 本次此线程共写入" + strconv.Itoa(writerNum) + "次\n")
			err := writeLog([]byte(str), mt)
			if err != nil {
				fmt.Println("write err")
			}
			isBreak = true
			break
		case receiveInfo := <-workCommand:
			fmt.Println("写入成功，线程id ", threadId)
			//计算间隔的ms 然后记录当前时间为上一次 发送等待下次请求
			intervalTime := receiveInfo.nowTime.UnixMilli() - lastTime.UnixMilli()
			lastTime = receiveInfo.nowTime
			intervalTimeFloat := float64(intervalTime)
			lastSeconds := intervalTimeFloat / 1000
			str := fmt.Sprintf(receiveInfo.nowTime.String() + " 线程id" + strconv.Itoa(threadId) + " 距离上次此协程发送" +
				strconv.FormatFloat(lastSeconds, 'f', -1, 64) + "秒\n")
			err := writeLog([]byte(str), mt)
			if err != nil {
				return
			}
			writerNum++
		}
		if isBreak {
			break
		}
	}
}

//以下为单独开启写log 上述为根据键盘输入线程数写log
func writeOnelog(wg *sync.WaitGroup, threadId string, mt *sync.Mutex) {
	var lastTime int64
	defer wg.Done()
	rand.Seed(time2.Now().UnixNano())
	for {
		key := rand.Intn(4)
		timeMs := timeArr[key]
		time2.Sleep(time2.Millisecond * timeMs)
		mt.Lock()
		fl, err := os.OpenFile(logAddr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			mt.Unlock()
			fmt.Println("open file err")
			wg.Done()
			return
		}
		time := time2.Now().UnixMicro()
		intervalTime := time - lastTime
		str := fmt.Sprintf(strconv.FormatInt(time, 10) + "|" + threadId + "|" + strconv.FormatInt(intervalTime, 10) + "\n")
		lastTime = time
		_, err = fl.Write([]byte(str))
		if err != nil {
			fmt.Println("write err")
		}
		mt.Unlock()
	}
}
