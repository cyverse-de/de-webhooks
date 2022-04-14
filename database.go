package main

import (
	"context"
	"database/sql"

	"github.com/cyverse-de/dbutil"
	"github.com/cyverse-de/queries"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
)

//Init init database connection
func Init() *sql.DB {
	dburi := config.GetString("db.uri")
	connector, err := dbutil.NewDefaultConnector("1m")
	if err != nil {
		Log.Fatal(err)
	}

	db, err := connector.Connect("postgres", dburi)
	if err != nil {
		Log.Fatal(err)
	}

	Log.Println("Connected to the database.")

	if err = db.Ping(); err != nil {
		Log.Fatal(err)
	}

	Log.Println("Successfully pinged the database.")
	return db
}

//getTemplates get template for given webhooks type e.g: slack
func (s *DBConnection) getTemplates(ctx context.Context) (map[string]string, error) {
	ctx, span := otel.Tracer(otelName).Start(ctx, "getTemplates")
	defer span.End()
	var id, temptext string
	tempmap := make(map[string]string)
	query := `select id, template from webhooks_type;`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&id, &temptext)
		if err != nil {
			return nil, err
		}
		tempmap[id] = temptext
	}
	if err := rows.Err(); err != nil {
		return tempmap, err
	}
	return tempmap, nil
}

//getUserInfo get User id for given user name
func (s *DBConnection) getUserInfo(ctx context.Context, username string) (string, error) {
	userID, err := queries.UserID(ctx, s.db, username)
	if err != nil {
		return "", err
	}
	return userID, nil
}

//getUserSubscriptions get user subscriptions to webhooks
func (s *DBConnection) getUserSubscriptions(ctx context.Context, uid string) ([]Subscription, error) {
	ctx, span := otel.Tracer(otelName).Start(ctx, "getUserSubscriptions")
	defer span.End()
	subs := []Subscription{}
	query := `select id, url, type_id from webhooks where user_id=$1`
	rows, err := s.db.QueryContext(ctx, query, string(uid))
	if err != nil {
		return subs, err
	}
	defer rows.Close()
	for rows.Next() {
		var sub Subscription
		err := rows.Scan(&sub.id, &sub.url, &sub.templatetype)
		if err != nil {
			return subs, err
		}
		topics, err := s.getTopics(ctx, sub.id)
		if err != nil {
			return subs, err
		}
		sub.topics = topics
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return subs, err
	}

	return subs, nil
}

func (s *DBConnection) getTopics(ctx context.Context, id string) ([]string, error) {
	topics := []string{}

	topicsquery := `select wt.topic from webhooks_topic as wt
	join webhooks_subscription as ws on wt.id = ws.topic_id
	where ws.webhook_id =$1`

	rows, err := s.db.QueryContext(ctx, topicsquery, string(id))
	if err != nil {
		return topics, err
	}
	defer rows.Close()
	for rows.Next() {
		var tp string
		err := rows.Scan(&tp)
		if err != nil {
			return topics, err
		}
		Log.Printf("Topic found: %s", tp)
		topics = append(topics, tp)
	}
	if err := rows.Err(); err != nil {
		return topics, err
	}
	return topics, nil
}
