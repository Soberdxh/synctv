package model

import "time"

type Current struct {
	Movie  CurrentMovie `json:"movie"`
	Status Status       `json:"status"`
}

type CurrentMovie struct {
	ID      string `json:"id,omitempty"`
	IsLive  bool   `json:"isLive,omitempty"`
	SubPath string `json:"subPath,omitempty"`
}

type Status struct {
	LastUpdate   time.Time `json:"lastUpdate,omitempty"`
	CurrentTime  float64   `json:"currentTime,omitempty"`
	PlaybackRate float64   `json:"playbackRate,omitempty"`
	IsPlaying    bool      `json:"isPlaying,omitempty"`
}

func NewStatus() Status {
	return Status{
		CurrentTime:  0,
		PlaybackRate: 1.0,
		LastUpdate:   time.Now(),
	}
}

func (c *Current) UpdateStatus() Status {
	if c.Movie.IsLive {
		c.Status.LastUpdate = time.Now()
		return c.Status
	}

	if c.Status.IsPlaying {
		c.Status.CurrentTime += time.Since(c.Status.LastUpdate).Seconds() * c.Status.PlaybackRate
	}

	c.Status.LastUpdate = time.Now()

	return c.Status
}

func (c *Current) setLiveStatus() Status {
	c.Status.IsPlaying = true
	c.Status.PlaybackRate = 1.0
	c.Status.CurrentTime = 0
	c.Status.LastUpdate = time.Now()

	return c.Status
}

// SeekBackwardTolerance 是进度退步容忍度(秒)。
// 正常时钟漂移 <1s,3s 能吸收测量噪声,同时拦截 buffer 卡顿/后台节流导致的明显退步。
const SeekBackwardTolerance = 3.0

func (c *Current) SetStatus(playing bool, seek, rate, timeDiff float64) Status {
	if c.Movie.IsLive {
		return c.setLiveStatus()
	}

	// 先外推当前进度到"现在",作为单调前进的基准
	c.UpdateStatus()

	// 计算客户端报的新进度(含时延补偿)
	var newTime float64
	if playing {
		newTime = seek + (timeDiff * rate)
	} else {
		newTime = seek
	}

	// 单调前进保护:若新进度明显落后于外推值,保留外推值,只更新播放状态/倍速。
	// 这样任一客户端因 buffer 卡顿/后台节流导致的落后进度,不会把服务器 CurrentTime 拉回并广播给全员。
	if newTime < c.Status.CurrentTime-SeekBackwardTolerance {
		c.Status.IsPlaying = playing
		c.Status.PlaybackRate = rate
		c.Status.LastUpdate = time.Now()
		return c.Status
	}

	// 正常前进或暂停:走原逻辑
	c.Status.IsPlaying = playing
	c.Status.PlaybackRate = rate
	if playing {
		c.Status.CurrentTime = seek + (timeDiff * rate)
	} else {
		c.Status.CurrentTime = seek
	}

	c.Status.LastUpdate = time.Now()

	return c.Status
}

func (c *Current) SetSeekRate(seek, rate, timeDiff float64) Status {
	if c.Movie.IsLive {
		return c.setLiveStatus()
	}

	if c.Status.IsPlaying {
		c.Status.CurrentTime = seek + (timeDiff * rate)
	} else {
		c.Status.CurrentTime = seek
	}

	c.Status.PlaybackRate = rate
	c.Status.LastUpdate = time.Now()

	return c.Status
}

func (c *Current) SetSeek(seek, timeDiff float64) Status {
	if c.Movie.IsLive {
		return c.setLiveStatus()
	}

	if c.Status.IsPlaying {
		c.Status.CurrentTime = seek + (timeDiff * c.Status.PlaybackRate)
	} else {
		c.Status.CurrentTime = seek
	}

	c.Status.LastUpdate = time.Now()

	return c.Status
}
