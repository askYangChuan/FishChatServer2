package etcd

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"github.com/golang/glog"
	"time"
)

// Master is a server
type Master struct {
	members  map[string]*Member
	etcCli   *clientv3.Client
	rootPath string
}

// Member is a client
type Member struct {
	InGroup bool
	IP      string
	Name    string
	CPU     int
}

func NewMaster(rootPath string, endpoints []string) (master *Master, err error) {
	var etcdClient *clientv3.Client
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second,
	}
	if etcdClient, err = clientv3.New(cfg); err != nil {
		glog.Error("Error: cannot connec to etcd:", err)
		return
	}
	master = &Master{
		members:  make(map[string]*Member),
		etcCli:   etcdClient,
		rootPath: rootPath,
	}
	return
}

func (m *Master) Members() (ms map[string]*Member) {
	ms = m.members
	return
}

func (m *Master) addWorker(info *WorkerInfo) {
	member := &Member{
		InGroup: true,
		IP:      info.IP,
		Name:    info.Name,
		CPU:     info.CPU,
	}
	m.members[member.Name] = member
}

func (m *Master) updateWorker(info *WorkerInfo) {
	member := m.members[info.Name]
	member.InGroup = true
}

func (m *Master) WatchWorkers() {
	rch := m.etcCli.Watch(context.Background(), m.rootPath, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			//fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			if ev.Type.String() == "EXPIRE" {
				member, ok := m.members[string(ev.Kv.Key)]
				if ok {
					member.InGroup = false
				}
			} else if ev.Type.String() == "PUT" {
				info := &WorkerInfo{}
				err := json.Unmarshal(ev.Kv.Value, info)
				if err != nil {
					glog.Error(err)
				}
				if _, ok := m.members[info.Name]; ok {
					m.updateWorker(info)
				} else {
					m.addWorker(info)
				}
			} else if ev.Type.String() == "DELETE" {
				delete(m.members, string(ev.Kv.Key))
			}
		}
	}
}
