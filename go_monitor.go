
package main

import (
	"bufio"
	"strconv"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"strings"
	"encoding/json"
	"io/ioutil"
    "net/http"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/disk"
	log "github.com/sirupsen/logrus"
)

type FsMoinitorInfo struct{
	DiskTotalSpace 				string  `json:"diskTotalSpace"`			//磁盘总体空间
	DiskUsableSpace				string	`json:"diskUsableSpace"`		//磁盘已用空间
	OpenFileDescriptorCount		string	`json:"openFileDescriptorCount"`//打开的文件描述符数	
	DiskCurrentUsage			string	`json:"diskCurrentUsage"`		//磁盘当前使用率
	TotalMemory					string	`json:"totalMemory"`			//内存总体数量	
	MemoryCurrentUsage			string	`json:"memoryCurrentUsage"`		//当前内存使用率,百分比
	Host_ip						string	`json:"ip"`						//主机ip
	Internal_port				string	`json:"internal_port"`			//sip内部服务端口
	External_port				string	`json:"external_port"`			//sip外部服务端口
	LogicalProcessorCount		string	`json:"logicalProcessorCount"`	//cpu逻辑数量
	OperateSystemVersion		string	`json:"operateSystemVersion"`	//操作系统版本
	CpuCurrentUsage				string	`json:"cpuCurrentUsage"`		//cpu当前使用lv
	CallTotalNum				string	`json:"callTotalNum"`			//当前呼叫总数量
	CallWaitingNum				string	`json:"callWaitingNum"`			//当前呼叫等待接续数
	CallInServiceNum			string	`json:"callInServiceNum"`		//当前坐席服务数量
	FsChildThreadNum            string  `json:"fsChildThreadNum"`       //记录freeswitch进程的线程数量
}

//var log = logrus.New()

func init() {

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)
	fsDir := strings.TrimSpace(os.Getenv(strings.ToUpper("fs_dir")))
	fileDir :=fmt.Sprintf("%s/log/monitor.log",fsDir)
	file, err := os.OpenFile(fileDir, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		// 设置将日志输出到文件
		log.SetOutput(file)
	} else {
		log.Info("打开日志文件失败，默认输出到stderr")
	}
}

//命令执行
//f
func Command(cmd string,f string)(result string,er error ){
	//c := exec.Command("cmd", "/C", cmd) 	// windows
	log.Infof("command cmd str :%s",cmd)
	c := exec.Command("bash", "-c", cmd)  // mac or linux
	stdout, err := c.StdoutPipe()
	if err != nil {
		result,er = "",err
		return
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader := bufio.NewReader(stdout)
		for {
			readString, err := reader.ReadString('\n')
			if err != nil || err == io.EOF {
				//result,er = "",err
				//fmt.Println("command return here1....")
				log.Infof("return here!1111,%s",readString)
				return
			}
			if len(f) == 0 {
				if readString != "" {
				   result = strings.Replace(readString, "\n", "", -1)
				   //fmt.Println(readString)
				   //fmt.Println("command return here2....")
				     
				}
				log.Infof("command readString :%s",readString)
			}else {
				if strings.Contains(readString,f){
					fmt.Println("return here3 result filter:",f);
					fmt.Println(readString)
					result = strings.Replace(readString, "\n", "", -1)
					
				} else {
					log.Infof("command return here 4")
				}
			}
		}
	}()
	err = c.Start()
	wg.Wait()
	fmt.Println("command result:",result)
	return 
}


//执行global_getvar local_ip_v4，获取IP
//ip为获取的值
func (fsinfo *FsMoinitorInfo)GetIp() error {
    //command 
	fsDir := strings.TrimSpace(os.Getenv(strings.ToUpper("fs_dir")))
	cmdStr :=fmt.Sprintf("%s/bin/fs_cli -x \"global_getvar local_ip_v4\"",fsDir)
	//found,err:=Command("/usr/local/freeswitch/bin/fs_cli -x \"global_getvar local_ip_v4\"","")
	found,err:=Command(cmdStr,"")
	if len(found) != 0 {
		fsinfo.Host_ip = found
		fmt.Println("get host ip :",found)
	}
	//fmt.Println("found----->",found)
	return err
}

//get sip port
func (fsinfo *FsMoinitorInfo)GetSipPort() error {
	fsDir := strings.TrimSpace(os.Getenv(strings.ToUpper("fs_dir")))
	cmdStr1 :=fmt.Sprintf("%s/bin/fs_cli -x \"global_getvar internal_sip_port\"",fsDir)
	//internal_port,err:=Command("/usr/local/freeswitch/bin/fs_cli -x \"global_getvar internal_sip_port\"","")
	internal_port,err:=Command(cmdStr1,"")
	if internal_port != "" {
		fsinfo.Internal_port = internal_port
	}
	cmdStr2 :=fmt.Sprintf("%s/bin/fs_cli -x \"global_getvar external_sip_port\"",fsDir)
	//external_port,err:=Command("/usr/local/freeswitch/bin/fs_cli -x \"global_getvar external_sip_port\"","")
	external_port,err:=Command(cmdStr2,"")
	if external_port != "" {
		fsinfo.External_port = external_port
	}
	return err
}

//获取cpu的数量和使用情况
func (fsinfo *FsMoinitorInfo)GetCpuInfo() error {
	c2, err:= cpu.Counts(false) //cpu物理核心
	if err != nil {
	   //fmt.Println("get cpu count error")
	   return err
	}
	fsinfo.LogicalProcessorCount = strconv.Itoa(c2)
	//获取cpu使用率
	cpuInfo, _ := cpu.Times(false)
	usageInfo :=(cpuInfo[0].User + cpuInfo[0].System)/(cpuInfo[0].Idle + cpuInfo[0].User + cpuInfo[0].System)
	fsinfo.CpuCurrentUsage = strconv.FormatFloat(float64(usageInfo)*100, 'f', 2, 32) + "%"
	return nil
}
//获取呼叫信息
//tcall:总呼叫量
//acall:接通坐席量
//wcall:排队的呼叫量
func (fsinfo *FsMoinitorInfo)GetCurrentCallInfo() error {
	fsDir := strings.TrimSpace(os.Getenv(strings.ToUpper("fs_dir")))
	cmdStr :=fmt.Sprintf("%s/bin/fs_cli -x \"show calls\"",fsDir)
	//totalCallStr,err:=Command("/usr/local/freeswitch/bin/fs_cli -x \"show calls\"","total")
	totalCallStr,err:=Command(cmdStr,"total")
	if totalCallStr != "" {
		//fsinfo.Internal_port = internal_port
		s := strings.Split(totalCallStr," ")
		fsinfo.CallTotalNum = s[0]	
	}
	cmdStr2 :=fmt.Sprintf("%s/bin/fs_cli -x \"show bridged_calls\"",fsDir)
	//bridgeCall,err:=Command("/usr/local/freeswitch/bin/fs_cli -x \"show bridged_calls\"","total")
	bridgeCall,err:=Command(cmdStr2,"total")
	if bridgeCall != "" {
		//fsinfo.Internal_port = internal_port
		b := strings.Split(bridgeCall," ")
		fsinfo.CallInServiceNum = b[0]
	}
	//获取两个数据的差值
	totalNum ,err := strconv.Atoi(fsinfo.CallTotalNum)
	if err != nil {
	   fmt.Println("conv total num error")
	   return err
	}
	bridgeCallNum,err := strconv.Atoi(fsinfo.CallInServiceNum)
	if err != nil {
	   fmt.Println("conv bridgeCallNum error")
	   return err
	}
	//获取等待的数据...
	waitNum := totalNum - bridgeCallNum;
	fsinfo.CallWaitingNum = strconv.Itoa(waitNum)	
	return err
}

//获取os的版本
func (fsinfo *FsMoinitorInfo)GetOsVersion() error {
	osVer,err:=Command("/usr/bin/cat /etc/redhat-release","")
	if osVer != "" {
	   fsinfo.OperateSystemVersion = osVer
	}
	return err
}
//对存储空间进行格式化
func StorageFormat(size float64) string {
	if size/(1024*1024*1024) > 1 {
		return strconv.FormatFloat(float64(size)/(1024*1024*1024), 'f', 2, 32) + "G"
	} else if size/(1024*1024) > 1 {
	    return strconv.FormatFloat(float64(size)/(1024*1024), 'f', 2, 32) + "M"
	} else if size/1024 > 1 {
	    return strconv.FormatFloat(float64(size)/(1024), 'f', 2, 32) + "K"
	} else {
		return strconv.FormatFloat(float64(size), 'f', 2, 32) + "B"
	}
}

//获取内存、使用情况
func (fsinfo *FsMoinitorInfo)GetMemoryUsed() error {
   memoryInfo, err := mem.VirtualMemory()
   memTotal := memoryInfo.Total
   memUsedPer := memoryInfo.UsedPercent 
   fmt.Println("----->memUsedPer:<-----",memUsedPer)
   //转化成字符串...
   fsinfo.MemoryCurrentUsage = strconv.FormatFloat(float64(memUsedPer), 'f', 2, 32) + "%"//保留两位小数 
   //fsinfo.TotalMemory = strconv.FormatFloat(float64(memTotal), 'f', 0, 32)//不需要保留小数...
   fsinfo.TotalMemory = StorageFormat(float64(memTotal))
   return err
}

//获取磁盘信息
func (fsinfo *FsMoinitorInfo)GetDiskInfo() error {
  info6, _ := disk.Usage("/") //指定某路径的硬盘使用情况
  //fsinfo.DiskTotalSpace = strconv.FormatFloat(float64(info6.Total), 'f', 0, 32) 
  //fsinfo.DiskUsableSpace = strconv.FormatFloat(float64(info6.Used), 'f', 0, 32)
  fsinfo.DiskTotalSpace = StorageFormat(float64(info6.Total))
  fsinfo.DiskUsableSpace = StorageFormat(float64(info6.Used))
  fsinfo.DiskCurrentUsage = strconv.FormatFloat(float64(info6.UsedPercent), 'f', 2, 32) + "%"
  return nil
}

//读取所有字符
func CommandReadAll(result *string,cmdStr string) {
    
	log.Infof("CommandReadAll str:%s",cmdStr)
	cmd := exec.Command("/bin/bash", "-c", cmdStr)

    //创建获取命令输出管道
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Infof("Error:can not obtain stdout pipe for command:%s\n", err)
        return
    }

    //执行命令
    if err := cmd.Start(); err != nil {
        log.Infof("Error:The command is err,", err)
        return
    }

    //读取所有输出
    bytes, err := ioutil.ReadAll(stdout)
    if err != nil {
        log.Infof("ReadAll Stdout:", err.Error())
        return
    }
	*result = string(bytes) 
	//time.Sleep(time.Duration(2)*time.Second)
    log.Infof("stdout:\n\n %s,result:%s", bytes,*result)
    if err := cmd.Wait(); err != nil {
        log.Infof("wait: %s", err.Error())
        return
    }
    log.Infof("stdout:\n\n %s", bytes)
}

//获取句柄使用情况
func (fsinfo *FsMoinitorInfo)GetHandlerInfo() error {
  //获取freeswitch进程的进程id
  var cmdRes string
  //pid,err:=Command("ps -ef|grep freeswitch|grep -v supervise|grep -v grep|awk '{print $2}'","")
  pid,err:=Command("pgrep freeswitch","")
  log.Infof("pid id :%s", pid)
  if pid != "" {
    //lsof -n|awk '{print $2}'|sort|uniq -c|sort -nr|more|grep
	//cmdStr := fmt.Sprintf("lsof -n|%s '{print $%d}'|%s|%s -c|%s -nr|%s|%s %s","awk",2,"sort","uniq","sort","more","grep",pid)
	cmdStr := "cat /proc/sys/fs/file-nr"
	log.Infof("cmd:%s",cmdStr)
	//handlerId,_ := Command(cmd_str,pid)
	//CommandReadAll(&cmdRes,cmd_str)
	//out, err1 := exec.Command("bash", "-c", cmd_str).Output()
	//log.Infof("proce id  %s:", string(out))
	//cmdStr := fmt.Sprintf("ls -l %s | head -n %d", ".", 10)
    cmd := exec.Command("sh", "-c", cmdStr)
    if out, err := cmd.CombinedOutput(); err != nil {
        log.Infof("Error: %v\n", err)
    } else {
        log.Infof("Success: %s\n%s\n", cmdStr, out)
		cmdRes = string(out)
    }	
	if cmdRes != "" {
		fmt.Println("handler Id:",cmdRes)
		hId := strings.TrimSpace(cmdRes)
		log.Infof("handler id %v:", hId)
		fsinfo.OpenFileDescriptorCount = strings.Split(hId,"\t")[0]
	}
  }
  return err
}

func (fsinfo *FsMoinitorInfo)SentPostFsMonitorInfo(url string) error{
    if "" == url {
      return nil
    }
   
    post,err := json.Marshal(*fsinfo)
    if err != nil {
        fmt.Println("Umarshal failed:", err)
        return err
    }
	var jsonStr = []byte(post)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}

    req.Header.Set("Content-Type", "application/json")
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		log.Infof("http post %s is error",url)
        panic(err)
    }
    defer resp.Body.Close()

    log.Infof("response Status:%s", resp.Status)
    log.Infof("response Headers:%v", resp.Header)
    body, _ := ioutil.ReadAll(resp.Body)
    log.Infof("response Body:%v", string(body))
	return nil
}

func (fsinfo *FsMoinitorInfo)GetFsChildThreadNum() error{
    pid,err:=Command("pgrep freeswitch","")
	log.Infof("pid id :%s", pid)
	if pid !="" {
		strCmd := fmt.Sprintf("/usr/bin/cat /proc/%s/status",pid)
		fsThreadNum,_:=Command(strCmd,"Threads:")
		log.Infof("thread command result:%s",fsThreadNum)
		hId := strings.TrimSpace(fsThreadNum)
		fsinfo.FsChildThreadNum = strings.Split(hId,"\t")[1]
	}
	return err
}

func main() {
	fscontext  := FsMoinitorInfo{
	DiskTotalSpace 				:"",
	DiskUsableSpace				:"",
	OpenFileDescriptorCount		:"",
	DiskCurrentUsage			:"",
	TotalMemory					:"",
	MemoryCurrentUsage			:"",
	Host_ip						:"",
	Internal_port				:"",
	External_port				:"",
	LogicalProcessorCount		:"",
	OperateSystemVersion		:"",
	CpuCurrentUsage				:"",
	CallTotalNum				:"",
	CallWaitingNum				:"",
	CallInServiceNum			:"",
	FsChildThreadNum            :"",
  }
  
  //var url string ="http://10.21.19.17:8888/test"
  //var url string = "http://10.21.19.11:80/config_manage/outToConfig/sendFSSystemInfo"
	url := strings.TrimSpace(os.Getenv(strings.ToUpper("http_url")))
	log.Infof("http url is :%s",url)
	if url == "" {
		log.Infof("http url is NULL")
		return
	}
  //调用方法
  fscontext.GetIp()
  fscontext.GetSipPort()
  fscontext.GetCpuInfo()
  fscontext.GetCurrentCallInfo()
  fscontext.GetOsVersion()
  fscontext.GetMemoryUsed()
  fscontext.GetDiskInfo()
  fscontext.GetHandlerInfo()
  fscontext.GetFsChildThreadNum()
  //通过http发送出去
  fscontext.SentPostFsMonitorInfo(url)
  log.Infof("http invoke finish......")
  
}



