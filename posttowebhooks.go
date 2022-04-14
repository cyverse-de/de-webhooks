package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

//compltedstatus Analysis completed status
const compltedstatus = "Completed"
const failedstatus = "Failed"

var httpClient = http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

//Payload payload to post to the webhooks
type Payload struct {
	ID, Name, Msg, Link, LinkText, Type string
	Completed                           bool
}

//Subscription defines user subscriptions to webhooks
type Subscription struct {
	id, templatetype, url string
	topics                []string
}

//template cache
var templatesmap map[string]string

func InitTemplatesMap(ctx context.Context, d *DBConnection) error {
	temmap, err := d.getTemplates(ctx)
	if err != nil {
		return err
	}
	templatesmap = temmap
	return nil
}

//ProcessMessages process the received message for post to webhooks
func ProcessMessage(ctx context.Context, d *DBConnection, del amqp.Delivery) error {
	if templatesmap == nil { // call only when template cache is not ready
		err := InitTemplatesMap(ctx, d)
		if err != nil {
			return err
		}
	}
	Log.Printf("[X] Notification %s", del.Body)
	uid := getUserID(ctx, d, del.Body)
	if uid != "" {
		return postToHook(ctx, d, uid, del.Body)
	} else {
		return errors.New("User not found")
	}
}

//getUserID Get user id for this Notification
func getUserID(ctx context.Context, d *DBConnection, msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "user")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("user is %s", string(value))
	uid, err := d.getUserInfo(ctx, string(value)+"@"+config.GetString("user.suffix"))
	if err != nil {
		Log.Error(err)
		return ""
	}
	return uid
}

//post to webhooks
func postToHook(ctx context.Context, d *DBConnection, uid string, msg []byte) error {
	ctx, span := otel.Tracer(otelName).Start(ctx, "postToHook")
	defer span.End()
	subs, err := d.getUserSubscriptions(ctx, uid)
	if err != nil {
		return err
	}
	Log.Printf("No. of subscriptions found: %d", len(subs))
	if len(subs) > 0 {
		for _, v := range subs {
			if isNotificationInTopic(msg, v.topics) {
				payload := preparePayloadFromTemplate(ctx, templatesmap[v.templatetype], msg)
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.url, payload)
				if err != nil {
					Log.Printf("Error posting to hook %s", err)
				}
				req.Header.Set("content-type", "application/json")
				resp, err := httpClient.Do(req)
				if err != nil {
					Log.Printf("Error posting to hook %s", err)
				}
				defer resp.Body.Close()
			}
		}
	}
	return nil
}

//isNotificationInTopic check if user is subscribed to this notification topic
func isNotificationInTopic(msg []byte, topics []string) bool {
	value, _, _, err := jsonparser.Get(msg, "message", "type")
	if err != nil {
		Log.Error(err)
		return false
	}

	if len(topics) < 1 {
		return false
	}

	for _, to := range topics {
		if string(value) == to {
			Log.Printf("Subscription topic found: %s", to)
			return true
		}
	}
	return false

}

//Prepare payload from template
func preparePayloadFromTemplate(ctx context.Context, templatetext string, msg []byte) *strings.Reader {
	_, span := otel.Tracer(otelName).Start(ctx, "preparePayloadFromTemplate")
	defer span.End()
	var buf1 bytes.Buffer
	var postbody Payload
	if len(templatetext) == 0 {
		payload := string(msg)
		Log.Printf("Empty Template. message to post: %s", payload)
		return strings.NewReader(payload)
	}
	t := template.Must(template.New("newtemplate").Parse(templatetext))
	w := io.MultiWriter(&buf1)
	isCompleted := (getType(msg) == "analysis") && isAnalysisCompleted(msg)
	postbody = Payload{ID: getID(msg),
		Msg:      getMessage(msg),
		Name:     getName(msg),
		Type:     getType(msg),
		Link:     config.GetString("de.base") + "/data/ds" + getResultFolder(msg),
		LinkText: "Go to results folder in DE", Completed: isCompleted}
	err := t.Execute(w, postbody)
	if err != nil {
		Log.Errorf("Error executing template: %s", err.Error())
	}
	Log.Printf("message to post: %s", buf1.String())
	return strings.NewReader(buf1.String())
}

//check if it is an analysis notification
func getType(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "type")
	if err != nil {
		Log.Error(err)
		return ""
	}
	return string(value)
}

//check if the analysis is completed
func isAnalysisCompleted(msg []byte) bool {
	Log.Printf("Getting analysis status")
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "analysisstatus")
	if err != nil {
		Log.Error(err)
	}
	Log.Printf("Analysis status is %s", value)
	if string(value) == compltedstatus || string(value) == failedstatus {
		return true
	}
	return false
}

//get analysis result folder
func getResultFolder(msg []byte) string {
	Log.Printf("Getting result folder")
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "analysisresultsfolder")
	if err != nil {
		Log.Error(err)
	}
	Log.Printf("Analysis result folder is %s", value)
	return string(value)
}

//get message from notfication
func getMessage(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "message", "text")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("Message is %s", value)
	return string(value)
}

//get id from notification
func getID(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "app_id")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("id is %s", value)
	return string(value)
}

//get name from notification
func getName(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "name")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("name is %s", value)
	return string(value)
}
