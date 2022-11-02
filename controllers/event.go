package controllers

import (
	codev1 "code/api/v1"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

func Handle(e <-chan CodeEvent, c client.Client, logger logr.Logger) {
	t := time.Tick(time.Second * 3)
	for {
		select {
		case ev := <-e:
			addorupdate(ev, logger)
		case <-t:
			updatestatus(c, logger)
		}
	}
}

func addorupdate(c CodeEvent, logger logr.Logger) {
	log := logger.WithName("resource add or update")
	l.Lock()
	defer l.Unlock()
	if obj, ok := m[c.resource.String()]; ok {
		if c.flag {
			log.Info(fmt.Sprintf("添加了%d秒\n", c.duration))
			obj.duration = obj.duration + c.duration
		}
		m[c.resource.String()] = obj
	} else {
		log.Info(fmt.Sprintf("resource: %s ,%d秒后过期,创建时间%s\n", c.resource.String(), c.duration, c.t.Time.Format("2006-01-02 15:04:05")))
		m[c.resource.String()] = c
	}
}

func updatestatus(c client.Client, logger logr.Logger) {
	log := logger.WithName("resource update status")
	for k, v := range m {
		t := v.t.Time
		n := time.Now()
		if n.Sub(t).Seconds() > float64(v.duration) {
			log.Info(fmt.Sprintf("删除时间%s\n", n.Format("2006-01-02 15:04:05")))
			code := &codev1.CodeServer{}
			if err := c.Get(context.TODO(), v.resource, code); err != nil {
				return
			}
			code.Status.Conditions = append(code.Status.Conditions, codev1.ServerCondition{
				Type:               codev1.ServerRecycled,
				Status:             corev1.ConditionTrue,
				Reason:             "the resource has expired.",
				Message:            nil,
				LastUpdateTime:     metav1.Time{},
				LastTransitionTime: metav1.Time{},
			})

			err := c.Status().Update(context.TODO(), code)
			if err != nil {
				log.Info("update ServerRecycled err")
			}
			delete(m, k)
		}
	}
}
