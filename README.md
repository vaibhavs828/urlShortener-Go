# urlShortener-Go
Implemented a URL Shortener Service: Developed a URL shortener application in Go that allows users to create and manage short URLs, with support for custom short URLs.

Integrated Caching and Message Queue: Used Redis to cache short URLs for quick retrieval and RabbitMQ for collecting and processing analytics data related to URL usage.

Concurrency Management: Implemented thread-safe operations with mutex locks to ensure consistent URL mapping in a concurrent environment.