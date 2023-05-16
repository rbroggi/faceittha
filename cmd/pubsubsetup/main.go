package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
)

func main() {
	// Check if the command-line argument is provided
	if len(os.Args) < 2 {
		fmt.Println("Please provide the comma-separated array as a command-line argument, following the patter: PROJECID,TOPIC1:SUBSCRIPTION11,SUBSCRIPTION12,TOPIC2:SUBSCRIPTION21:SUBSCRIPTION22")
		return
	}

	// Get the comma-separated array from the command-line argument
	arrayStr := os.Args[1]

	// Split the comma-separated array into individual items
	items := strings.Split(arrayStr, ",")
	projectID := strings.ReplaceAll(items[0], " ", "")
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Panicf("Unable to create client to project %q: %s", projectID, err)
	}
	defer client.Close()
	fmt.Println("Project ID:", projectID)
	items = items[1:]

	// Process each item
	for _, item := range items {
		// Split the item into topic and subscriptions
		parts := strings.Split(item, ":")
		topicID := strings.ReplaceAll(parts[0], " ", "")
		topic, err := client.CreateTopic(ctx, topicID)
		if err != nil && !strings.Contains(err.Error(), "Topic already exists") {
			log.Panicf("Unable to create topic %s for project %s: %v", topicID, projectID, err)
		} else if err != nil && strings.Contains(err.Error(), "Topic already exists") {
			topic = client.Topic(topicID)
		}
		subscriptions := parts[1:]

		for _, s := range subscriptions {
			subscriptionID := strings.ReplaceAll(s, " ", "")

			_, err = client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{Topic: topic})
			if err != nil && !strings.Contains(err.Error(), "Subscription already exists") {
					log.Panicf("Unable to create subscription %s on topic %s for project %s: %v", subscriptionID, topicID, projectID, err)
			}
			fmt.Printf("Project, topic, subscription: [%s, %s, %s]\n", projectID, topic, subscriptionID)
		}

	}
}
