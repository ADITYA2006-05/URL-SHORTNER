# Distributed URL Shortener 🔗

A **high-performance distributed URL shortening service** built for scalability, reliability, and low-latency redirection. This project demonstrates real-world **system design**, **backend engineering**, and **distributed systems** concepts by handling URL shortening at scale.

---

## 📌 Project Purpose

Modern applications generate and share millions of URLs daily. Traditional URL shorteners struggle with:

* High traffic loads
* Database bottlenecks
* Slow redirection
* Scalability issues
* Single points of failure

This project solves these challenges by building a **distributed architecture** that can process and redirect millions of requests efficiently.

---

## 🎯 Why This Project Adds Weight to Your Resume

This project showcases skills recruiters highly value:

✅ **System Design Knowledge** – scalable architecture design
✅ **Distributed Systems** – handling services across multiple nodes
✅ **Caching Expertise** – optimized using Redis
✅ **Backend Engineering** – API development in Go
✅ **Scalability Concepts** – horizontal scaling & load balancing
✅ **Performance Optimization** – low latency responses
✅ **Production-Level Thinking** – fault tolerance & reliability

**Resume Impact: 9.5/10 ⭐**

---

## 🚀 Features

* Generate unique short URLs
* Custom short links support
* High-speed URL redirection
* Click analytics (track visits)
* URL expiration handling
* Redis-based caching
* Distributed architecture
* Load balancing support
* Fault tolerant design
* RESTful APIs

---

## 🛠 Tech Stack

**Backend**

* Go (Golang)

**Cache**

* Redis

**Containerization**

* Docker

**API Testing**

* [Postman](https://www.postman.com/?utm_source=chatgpt.com)

**Version Control**

* [GitHub](https://github.com/?utm_source=chatgpt.com)

---

## 🏗 System Architecture

```text
Client Request
      ↓
Load Balancer
      ↓
Multiple Go API Servers
      ↓
Redis Cache Layer
      ↓
Persistent Database
```

### Flow:

1. User submits long URL.
2. Service generates unique short code.
3. Mapping stored in Redis/database.
4. User accesses short URL.
5. Service redirects instantly.

---

## 📂 Project Structure

```bash
distributed-url-shortener/
│── cmd/
│── internal/
│   ├── handlers/
│   ├── services/
│   ├── models/
│   └── config/
│── docker/
│── redis/
│── main.go
│── Dockerfile
│── docker-compose.yml
│── README.md
```

---

## ⚙️ Installation

### Clone repository

```bash
git clone https://github.com/yourusername/distributed-url-shortener.git
cd distributed-url-shortener
```

### Build Docker containers

```bash
docker-compose up --build
```

### Run service

```bash
localhost:8080
```

---

## API Endpoints

### Create Short URL

```http
POST /shorten
```

Request:

```json
{
  "url": "https://example.com"
}
```

Response:

```json
{
  "short_url": "http://localhost:8080/a1b2c3"
}
```

---

### Redirect

```http
GET /{short_code}
```

Redirects to original URL.

---

### Analytics

```http
GET /analytics/{short_code}
```

Returns:

* total clicks
* creation date
* expiry date

---

## Scaling Strategies Implemented

* Horizontal scaling with multiple Go instances
* Redis caching to reduce DB hits
* Stateless API design
* Load balancing ready
* Microservice-friendly design

---

## Future Enhancements

* User authentication
* QR code generation
* Custom domains
* Rate limiting
* Kubernetes deployment
* Monitoring dashboard

---

## Learning Outcomes

By building this project, you demonstrate understanding of:

* Distributed Systems
* Caching Strategies
* API Design
* Scalability
* Load Balancing
* Containerization
* Production Deployment

---

## 👨‍💻 Author

**Aditya**
[GitHub Profile](https://github.com/ADITYA2006-05?utm_source=chatgpt.com)

---

## 📜 License

MIT License.
