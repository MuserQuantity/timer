package timer

import (
	"log"
	"sort"
	"sync"
	"time"
)

// 每日定点定时器
type DailyFixedTimer map[string][3]int //map["routine_1"][3]int{"24h","m","s"}

func (self DailyFixedTimer) Wait(routine string) {
	tdl := self.deadline(routine)
	log.Printf("************************ ……<%s> 每日定时器等待至 %v ……************************\n", routine, tdl.Format("2006-01-02 15:04:05"))
	time.Sleep(tdl.Sub(time.Now()))
}

func (self DailyFixedTimer) deadline(routine string) time.Time {
	t := time.Now()
	if t.Hour() > self[routine][0] {
		t = t.Add(24 * time.Hour)
	} else if t.Hour() == self[routine][0] && t.Minute() > self[routine][1] {
		t = t.Add(24 * time.Hour)
	} else if t.Hour() == self[routine][0] && t.Minute() == self[routine][1] && t.Second() >= self[routine][2] {
		t = t.Add(24 * time.Hour)
	}
	year, month, day := t.Date()
	return time.Date(year, month, day, self[routine][0], self[routine][1], self[routine][2], 0, time.Local)
}

// 动态倒计时器
type CountdownTimer struct {
	// 倒计时的时间(min)级别，由小到大排序
	Level []float64
	// 倒计时对象的非正式计时表
	Routines map[string]float64
	//更新标记
	Flag map[string]chan bool
	sync.RWMutex
}

func NewCountdownTimer(level []float64, routine []string) *CountdownTimer {
	if len(level) == 0 {
		level = []float64{60 * 24}
	}
	sort.Float64s(level)
	ct := &CountdownTimer{
		Level:    level,
		Routines: make(map[string]float64),
		Flag:     make(map[string]chan bool),
	}
	for _, v := range routine {
		ct.Routines[v] = ct.Level[0]
	}
	return ct
}

func (self *CountdownTimer) Wait(routine string) {
	self.RWMutex.RLock()
	defer self.RWMutex.RUnlock()
	if _, ok := self.Routines[routine]; !ok {
		return
	}
	self.Flag[routine] = make(chan bool)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("动态倒计时器: %v\n", err)
		}
		select {
		case <-self.Flag[routine]:
			self.Routines[routine] = self.Routines[routine] / 1.2
			if self.Routines[routine] < self.Level[0] {
				self.Routines[routine] = self.Level[0]
			}
		default:
			self.Routines[routine] = self.Routines[routine] * 1.2
			if self.Routines[routine] > self.Level[len(self.Level)-1] {
				self.Routines[routine] = self.Level[len(self.Level)-1]
			}
		}
	}()
	for k, v := range self.Level {
		if v < self.Routines[routine] {
			continue
		}

		if k != 0 && v != self.Routines[routine] {
			k--
		}
		log.Printf("************************ ……<%s> 倒计时等待 %v 分钟……************************\n", routine, self.Level[k])
		time.Sleep(time.Duration(self.Level[k]) * time.Minute)
		break
	}
	close(self.Flag[routine])
}

func (self *CountdownTimer) Update(routine string) {
	go func() {
		defer func() {
			recover()
		}()
		select {
		case self.Flag[routine] <- true:
		default:
			return
		}
	}()
}

func (self *CountdownTimer) AddRoutine(routine string, t float64) *CountdownTimer {
	self.RWMutex.Lock()
	defer self.RWMutex.Unlock()
	self.Routines[routine] = t
	return self
}

func (self *CountdownTimer) ReSetLevel(level []float64) *CountdownTimer {
	self.RWMutex.Lock()
	defer self.RWMutex.Unlock()
	self.Level = level
	return self
}
