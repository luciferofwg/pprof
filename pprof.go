package pprof

import (
	"log"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/buaazp/fasthttprouter"
	"github.com/json-iterator/go"
	"github.com/kataras/iris/httptest"
	"github.com/valyala/fasthttp"
)

const (
	serviceName = "pprof"
)

var (
	dirFullPath string
	router      *fasthttprouter.Router = nil
	fCpu        *os.File               = nil
	fMem        *os.File               = nil
	fTrace      *os.File               = nil
	fBlock      *os.File               = nil
)

type msgResp struct {
	Cpu   string `json:"cpu"`
	Mem   string `json:"mem"`
	Trace string `json:"trace"`
	Block string `json:"block,omitempty"`
}

func init() {
	curPath, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	dirFullPath = path.Join(curPath, "pprof")
	if !isExists(dirFullPath) {
		if err := os.Mkdir(dirFullPath, 0666); err != nil {
			log.Panic(err)
		}
	}

	router = fasthttprouter.New()
	router.POST("/start", handleStart)
	router.GET("/start", handleStart)
	router.POST("/stop", handleStop)
	router.GET("/stop", handleStop)
	router.POST("/gc", handleGC)
	router.GET("/gc", handleGC)
}

func Pprof() {
	go func() {
		ser := &fasthttp.Server{
			Name:    serviceName,
			Handler: router.Handler,
		}
		if err := ser.ListenAndServe(":6060"); err != nil {
			log.Panic(err)
		}
	}()
}
func generateFile(filepath string) (*os.File, error) {
	f, err := os.Create(filepath)
	if err != nil {
		return &os.File{}, nil
	}
	return f, nil
}

func handleStart(ctx *fasthttp.RequestCtx) {
	log.Printf("recv profile start.")
	filepaths := []string{path.Join(dirFullPath, "cpu.pprof"), path.Join(dirFullPath, "mem.pprof"), path.Join(dirFullPath, "trace.pprof"), path.Join(dirFullPath, "block.pprof")}
	deleteFile(filepaths)
	var err error
	fCpu, err = generateFile(filepaths[0])
	if err != nil {
		log.Printf("generate cpu.pprof failed. %v")
		respErr(ctx, &msgResp{
			Cpu: "generate cpu.pprof failed.",
		})
		return
	}
	if err := pprof.StartCPUProfile(fCpu); err != nil {
		log.Printf("start cpu profile failed.%v", err)
		respErr(ctx, &msgResp{
			Cpu: "start cpu profile failed.",
		})
		return
	}

	fMem, err = generateFile(filepaths[1])
	if err != nil {
		log.Printf("generate mem.pprof failed. %v")
		respErr(ctx, &msgResp{
			Mem: "generate mem.pprof failed.",
		})
		return
	}
	if err := pprof.WriteHeapProfile(fMem); err != nil {
		log.Printf("start mem profile failed.%v", err)
		respErr(ctx, &msgResp{
			Mem: "start mem profile failed.",
		})
		return
	}

	fTrace, err = generateFile(filepaths[2])
	if err != nil {
		log.Printf("generate trace.pprof failed. %v")
		respErr(ctx, &msgResp{
			Trace: "generate mem.pprof failed.",
		})
		return
	}
	if err := trace.Start(fTrace); err != nil {
		log.Printf("start trace profile failed.%v", err)
		respErr(ctx, &msgResp{
			Trace: "start trace profile failed.",
		})
		return
	}

	fBlock, err = generateFile(filepaths[3])
	if err != nil {
		log.Printf("generate block.pprof failed. %v")
		respErr(ctx, &msgResp{
			Block: "generate block.pprof failed.",
		})
		return
	}
	if err := pprof.Lookup("block").WriteTo(fBlock, 0); err != nil {
		log.Printf("start block profile failed.%v", err)
		respErr(ctx, &msgResp{
			Block: "start block profile failed.",
		})
		return
	}

	respSucc(ctx, &msgResp{
		Cpu:   "cpu profile start success",
		Mem:   "mem profile start success",
		Trace: "trace profile start success",
		Block: "block profile start success",
	})
}

func handleStop(ctx *fasthttp.RequestCtx) {
	log.Printf("recv profile stop.")
	if fCpu != nil {
		pprof.StopCPUProfile()
		fCpu.Close()
	}

	// 内存不需要停止
	fMem.Close()
	if fTrace != nil {
		trace.Stop()
		fTrace.Close()
	}
	// block直接关闭
	fBlock.Close()

	respSucc(ctx, &msgResp{
		Cpu:   "cpu profile stop success",
		Mem:   "mem profile stop success",
		Trace: "trace profile stop success",
		Block: "block profile stop success",
	})
}

func handleGC(ctx *fasthttp.RequestCtx) {
	runtime.GC()
}

func respErr(ctx *fasthttp.RequestCtx, msg *msgResp) {
	ctx.Response.SetStatusCode(httptest.StatusInternalServerError)
	msgStr, _ := jsoniter.MarshalToString(msg)
	ctx.Response.SetBodyString(msgStr)
}

func respSucc(ctx *fasthttp.RequestCtx, msg *msgResp) {
	ctx.Response.SetStatusCode(httptest.StatusInternalServerError)
	msgStr, _ := jsoniter.MarshalToString(msg)
	ctx.Response.SetBodyString(msgStr)
}

/**
 * @Author      : Administrator
 * @Description : 判断目录是否存在
 * @Date        : 2020/7/2 0002 15:50
 * @Param       :
 * @return      : 存在：true；不存在：false
 **/
func isExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func deleteFile(filepaths []string) {
	for _, filepath := range filepaths {
		if isExists(filepath) {

			if err := os.Remove(filepath); err != nil {
				log.Printf("delete %v failed.%v", filepath, err)
			}
		}
	}
}
