package controllers

import (
	codev1 "code/api/v1"
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

type CodeEvent struct {
	duration int64
	resource types.NamespacedName
	t        metav1.Time
	flag     bool
}

var m = make(map[string]CodeEvent)
var l sync.Mutex

func Handle(e <-chan CodeEvent, c client.Client) {
	t := time.Tick(time.Second * 3)
	for {
		select {
		case ev := <-e:
			addorupdate(ev)
		case <-t:
			updatestatus(c)
		}
	}
}

func addorupdate(c CodeEvent) {
	l.Lock()
	defer l.Unlock()
	if obj, ok := m[c.resource.String()]; ok {
		if c.flag {
			log.Printf("添加了%d秒\n", c.duration)
			obj.duration = obj.duration + c.duration
		}
		m[c.resource.String()] = obj
	} else {
		log.Printf("%d秒后过期\n", c.duration)
		log.Printf("创建时间%s\n", c.t.Time.Format("2006-01-02 15:04:05"))
		m[c.resource.String()] = c
	}
}

func updatestatus(c client.Client) {
	for k, v := range m {
		t := v.t.Time
		n := time.Now()
		if n.Sub(t).Seconds() > float64(v.duration) {
			log.Printf("删除时间%s\n", n.Format("2006-01-02 15:04:05"))
			code := &codev1.CodeServer{}
			if err := c.Get(context.TODO(), v.resource, code); err != nil {
				return
			}
			code.Status.Conditions = append(code.Status.Conditions, codev1.ServerCondition{
				Type:               codev1.ServerRecycled,
				Status:             corev1.ConditionTrue,
				Reason:             "",
				Message:            nil,
				LastUpdateTime:     metav1.Time{},
				LastTransitionTime: metav1.Time{},
			})

			err := c.Status().Update(context.TODO(), code)
			if err != nil {
				log.Println("update ServerRecycled err")
			}
			delete(m, k)
		}
	}
}
