# kafka

Turns the Kafka topics in your landing zone into a resource your application can depend on.

Add it to the canvas, deploy. No settings. It emits a `kafka-authentication` resource: bootstrap brokers, SASL mechanism and credentials, the list of your topics, and the prefix your consumer groups must start with. Connect that to any application bundle with a Kafka port.

It creates nothing. The topics and the credential already exist; this bundle shapes them for an application.
