# kafka

{{#resources.kafka}}
Topics under `{{dependencies.landing_zone.kafka.topic_prefix}}`, consumer groups under `{{resources.kafka.consumer_group_prefix}}`. SASL {{resources.kafka.auth.mechanism}} over TLS.

## Check the credential from a laptop

```sh
kcat -b {{dependencies.landing_zone.kafka.bootstrap_brokers.0}} -L \
  -X security.protocol=SASL_SSL -X sasl.mechanisms={{resources.kafka.auth.mechanism}} \
  -X sasl.username={{resources.kafka.auth.username}} -X sasl.password=<password from the resource>
```

Only your topics appear in the listing. That is the credential working, not a missing topic.
{{/resources.kafka}}

## Authorization failed

The credential reaches topics under your prefix and consumer groups under your prefix, nothing else. A topic or group name outside those is denied by the broker. Check the application uses a topic from the resource's `topics` list and a consumer group that starts with `consumer_group_prefix`.

## Producer or consumer stalls

Each landing zone has a byte-rate quota. A client that hits it is throttled, not disconnected. Slow the producer or ask for the limit to be raised.
