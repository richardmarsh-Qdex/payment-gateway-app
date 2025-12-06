# Payment Gateway API - Go

A high-performance, secure payment gateway API built with Go, featuring comprehensive security measures, efficient database operations, and robust error handling.

## Features

### Security
- **JWT Authentication** - Secure token-based authentication
- **API Key Authentication** - Alternative authentication method for merchants
- **Data Encryption** - AES-GCM encryption for sensitive data (CVV, card numbers)
- **Rate Limiting** - Per-IP rate limiting to prevent abuse
- **Security Headers** - CORS, XSS protection, content security policy
- **Input Validation** - Comprehensive validation for all inputs
- **PCI Compliance Considerations** - Card data encryption and secure storage

### Performance
- **Connection Pooling** - Efficient database connection management
- **Async Processing** - Background processing for payment transactions
- **Indexed Queries** - Optimized database queries with proper indexing
- **Graceful Shutdown** - Clean server shutdown with connection cleanup

### Functionality
- **Multiple Payment Methods** - Card, Bank, Wallet, Crypto support
- **Payment Processing** - Complete payment lifecycle management
- **Refunds** - Full and partial refund support
- **Webhooks** - Event notifications (structure ready)
- **Merchant Management** - Merchant account creation and management
- **Transaction Tracking** - Comprehensive transaction history

## Prerequisites

- Go 1.23 or higher
- PostgreSQL 12 or higher
- Redis (optional, for caching)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd payment-gateway-go
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Set up PostgreSQL database:
```sql
CREATE DATABASE payment_gateway;
```

5. Run the application:
```bash
go run main.go
```

## Configuration

Key environment variables:

- `SERVER_PORT` - Server port (default: 8080)
- `DB_HOST` - Database host
- `DB_NAME` - Database name
- `JWT_SECRET_KEY` - JWT signing key (min 32 chars)
- `ENCRYPTION_KEY` - Data encryption key (min 32 chars)
- `RATE_LIMIT_RPS` - Rate limit requests per second

See `.env.example` for all configuration options.

## API Endpoints

### Authentication

#### Create Merchant
```http
POST /api/v1/merchants
Content-Type: application/json

{
  "name": "Merchant Name",
  "email": "merchant@example.com"
}
```

#### Generate Token
```http
POST /api/v1/merchants/token
Content-Type: application/json

{
  "api_key": "pk_...",
  "secret_key": "sk_..."
}
```

### Payments

#### Create Payment
```http
POST /api/v1/payments
X-API-Key: pk_...
Content-Type: application/json

{
  "amount": 100.00,
  "currency": "USD",
  "payment_method": "card",
  "customer_email": "customer@example.com",
  "customer_name": "John Doe",
  "card_number": "4111111111111111",
  "expiry_month": 12,
  "expiry_year": 2025,
  "cvv": "123",
  "cardholder_name": "John Doe",
  "description": "Payment for order #123"
}
```

#### Get Payment
```http
GET /api/v1/payments/{payment_id}
X-API-Key: pk_...
```

#### List Payments
```http
GET /api/v1/payments?limit=20&offset=0
X-API-Key: pk_...
```

#### Refund Payment
```http
POST /api/v1/payments/{payment_id}/refund
X-API-Key: pk_...
Content-Type: application/json

{
  "amount": 50.00,
  "reason": "Customer request"
}
```

### Health Checks

#### Health Check
```http
GET /health
```

#### Readiness Check
```http
GET /ready
```

## Security Best Practices

1. **Change Default Keys**: Always change `JWT_SECRET_KEY` and `ENCRYPTION_KEY` in production
2. **Use HTTPS**: Always use HTTPS in production
3. **Environment Variables**: Never commit `.env` files
4. **Database Security**: Use strong database passwords and SSL connections
5. **Rate Limiting**: Adjust rate limits based on your needs
6. **PCI Compliance**: For production, ensure PCI DSS compliance for card data handling

## Database Schema

The application uses GORM for database migrations. Tables are automatically created:

- `merchants` - Merchant accounts
- `payments` - Payment transactions
- `payment_cards` - Encrypted card details
- `refunds` - Refund transactions
- `webhooks` - Webhook delivery records

## Project Structure

```
payment-gateway-go/
├── config/          # Configuration management
├── database/        # Database connection and migrations
├── handlers/        # HTTP request handlers
├── middleware/      # HTTP middleware (auth, rate limiting, etc.)
├── models/          # Data models
├── services/        # Business logic
├── utils/           # Utility functions (encryption, validation)
├── main.go          # Application entry point
└── README.md        # This file
```

## Testing

Run tests:
```bash
go test ./...
```

## Performance Considerations

- Database connection pooling is configured for optimal performance
- Payment processing happens asynchronously to avoid blocking requests
- Rate limiting prevents abuse while allowing legitimate traffic
- Indexed database queries for fast lookups

## Production Deployment

1. Set `ENVIRONMENT=production` in `.env`
2. Use strong, randomly generated keys for `JWT_SECRET_KEY` and `ENCRYPTION_KEY`
3. Enable SSL for database connections (`DB_SSLMODE=require`)
4. Configure proper firewall rules
5. Set up monitoring and logging
6. Use a reverse proxy (nginx, Caddy) for HTTPS termination
7. Implement proper backup strategies

## License

[Your License Here]

## Contributing

[Contributing Guidelines Here]

