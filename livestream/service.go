package livestream

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nathanusask/ipfs-livestream/utils"

	"github.com/go-playground/lars"

	"github.com/pkg/errors"

	"github.com/nathanusask/ipfs-livestream/ffmpeg"
	"github.com/nathanusask/ipfs-livestream/ipfs"
)

const syncFile = "sync.json"

type service struct {
	Parts          []string      `json:"parts"`
	TempSample     string        `json:"-"`
	SampleCursor   int32         `json:"cursor"`
	SampleDuration time.Duration `json:"sample"`
	Ended          bool          `json:"ended"`
	Started        string        `json:"started"`
	Updated        string        `json:"updated"`

	dataFolder string `json:"-"`

	ipfsClient   ipfs.Interface   `json:"-"`
	ffmpegClient ffmpeg.Interface `json:"-"`

	_sync          int32  `json:"-"`
	_lastSync      int32  `json:"-"`
	_syncfileCache []byte `json:"-"`
}

func New(ffmpegPath, dataFolder string, sampleDuration time.Duration) Interface {
	return &service{
		Parts:          make([]string, 0),
		SampleDuration: sampleDuration,
		dataFolder:     dataFolder,
		ipfsClient:     ipfs.New(),
		ffmpegClient:   ffmpeg.New(ffmpegPath),
	}
}

func enableCors(c lars.Context) {
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Next()
}

func (s *service) watchSync(c lars.Context) {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Write(s._syncfileCache)
	c.Response().WriteHeader(http.StatusOK)
}

func (s *service) UseDefaultDevices() error {
	devices, err := s.ffmpegClient.GetAvailableDevices()
	if err != nil {
		return err
	}
	if len(devices.Video) < 1 || len(devices.Audio) < 1 {
		return errors.New("video or audio device is unavailable")
	}
	if runtime.GOOS == "windows" {
		s.ffmpegClient.SetDevices(devices.Video[0], devices.Audio[0])
	} else {
		s.ffmpegClient.SetDevices(strconv.Itoa(len(devices.Video)-1), "0")
	}
	return nil
}

func (s *service) SetDevices(videoDevice, audioDevice string) {
	s.ffmpegClient.SetDevices(videoDevice, audioDevice)
}

func (s *service) Watch(ipnsAddress string) error {
	var err error
	log.Println("reading the stream", ipnsAddress)

	router := lars.New()
	router.Use(enableCors)
	router.Get("/sync", s.watchSync)
	server := &http.Server{Addr: ":8888", Handler: router.Serve()}

	go server.ListenAndServe()
	defer server.Close()

	lastHash := ""
	for !s.Ended {
		log.Println("checking for updates...")
		syncPath := s.dataFolder + "/" + syncFile
		err = utils.IpnsDownloadFile(ipnsAddress, syncPath)
		if err != nil {
			return err
		}
		hash, err := utils.HashMD5(syncPath)
		if err != nil {
			return err
		}
		if hash == lastHash {
			log.Println("no updates from the streamer")
			time.Sleep(s.SampleDuration)
			continue
		}
		lastHash = hash
		data, err := ioutil.ReadFile(syncPath)
		if err != nil {
			return err
		}
		s._syncfileCache = data
		err = json.Unmarshal(data, s)
		if err != nil {
			return err
		}
		log.Println("stream updated. Now contains", len(s.Parts), "parts")
	}
	log.Println("stream ended")
	return nil
}

func (s *service) Broadcast(samples int) error {
	var err error
	err = utils.CreateDir(s.dataFolder)
	if err != nil {
		log.Fatalln("Failed to create directory ", s.dataFolder, " with error ", err)
		return err
	}
	id, err := s.ipfsClient.GetId()
	if err != nil {
		return err
	}
	log.Println("Broadcasting with ID", id.ID)
	s.Started = time.Now().String()
	i := 0
	for {
		if samples > 0 {
			i++
			if i > samples {
				syncCursor := atomic.LoadInt32(&s._sync)
				if syncCursor > 0 {
					s._lastSync = syncCursor
					log.Println("waiting for the synchronization to finish...")
					time.Sleep(time.Second * 5)
					continue
				} else if s._lastSync != s.SampleCursor {
					log.Println("running the final synchronization...")
					s.Ended = true
					s.safeSync()
				}
				return nil
			}
		}
		if s.SampleCursor > 0 {
			if !utils.FileExists(s.TempSample) {
				return errors.New("sample does not exist or was not recorded")
			}
			go s.pushSample(s.TempSample)
		}
		// record the screen
		s.TempSample = s.dataFolder + "/sample_" + strconv.Itoa(int(s.SampleCursor)) + ".mp4"
		log.Println("recording...", s.TempSample)
		err = s.recordSample()
		if err != nil {
			return err
		}
		s.SampleCursor++
	}
}

func (s *service) recordSample() error {
	return s.ffmpegClient.RecordScreen(s.TempSample, s.SampleDuration)
}

func (s *service) pushSample(tempSample string) {
	const tenSecond = time.Second * 10
	t := s.SampleDuration / 2
	if t > tenSecond {
		log.Println("preparing in", tenSecond)
		time.Sleep(tenSecond)
	} else {
		log.Println("preparing in", t)
		time.Sleep(t)
	}
	// adding to IPFS
	log.Println("uploading...")
	fn, err := s.ipfsClient.PushFile(tempSample)
	if err != nil {
		panic(err)
	}
	log.Println("added", fn)
	s.Parts = append(s.Parts, fn)
	// update the stream
	s.safeSync()
}

func (s *service) safeSync() {
	log.Println("synchronizing...")
	data, err := json.Marshal(s)
	if err != nil {
		log.Println("failed to encode the sync.json due", err.Error())
		return
	}
	err = ioutil.WriteFile(s.dataFolder+"/"+syncFile, data, os.ModePerm)
	if err != nil {
		log.Println("failed to write to sync.json due", err.Error())
		return
	}
	err = s.sync()
	if err != nil {
		log.Println("ERROR:", err.Error())
	}
}

func (s *service) sync() error {
	if atomic.LoadInt32(&s._sync) > 0 {
		log.Println("aborted. Awaiting for the previous synchronization to finish")
		return nil
	}
	s.Updated = time.Now().String()
	atomic.StoreInt32(&s._sync, s.SampleCursor)
	defer atomic.StoreInt32(&s._sync, 0)
	hash, err := s.ipfsClient.PushFile(s.dataFolder + "/" + syncFile)
	if err != nil {
		return err
	}
	err = s.ipfsClient.PublishName(hash)
	log.Println("synchronization is over for", hash)
	return err
}
