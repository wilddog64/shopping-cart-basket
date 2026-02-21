# Project Brief: Basket Service

## What This Project Does

The Basket Service is a Go microservice that manages shopping cart sessions for the Shopping Cart platform. It is the user-facing entry point for all cart interactions, providing a REST API for customers to build up a basket of products before initiating checkout.

## Core Responsibilities

- **Cart session management**: Create, retrieve, and persist shopping carts per customer in Redis with automatic TTL expiration (7 days default)
- **Item operations**: Add items to cart (with automatic quantity merging for duplicate products), update item quantities, remove individual items, and clear the entire cart
- **Total calculation**: Real-time recalculation of item subtotals and cart total on every mutation
- **Checkout orchestration**: Validate cart is non-empty, publish a `cart.checkout` event to RabbitMQ (which the Order Service consumes to create an order), then clear the cart
- **Event publishing**: Publish cart lifecycle events (`cart.created`, `cart.updated`, `cart.cleared`, `cart.checkout`) to RabbitMQ for downstream services

## Goals

- Provide a fast, low-latency cart API (target: p50 < 10ms, p99 < 50ms)
- Support horizontal scaling with stateless design (all state in Redis)
- Enable event-driven order creation via the checkout flow
- Enforce authentication for all cart operations (Keycloak JWT or dev X-User-ID header)

## Scope

**In scope:**
- Shopping cart CRUD per authenticated customer
- Cart TTL management
- RabbitMQ event publishing
- JWT authentication via Keycloak
- Prometheus metrics and structured logging
- Kubernetes-ready deployment

**Out of scope:**
- Order management (handled by shopping-cart-order service)
- Payment processing (handled by shopping-cart-payment service)
- Product catalog validation (product details are passed in by the client)
- Cart merging across devices/sessions

## Service Context in the Platform

This service sits at the beginning of the purchase funnel. The checkout flow triggers the Order Service via a `cart.checkout` RabbitMQ event. It reads product details directly from the client request (no synchronous call to the Product Catalog service).

## Status

**In Development** — core functionality is implemented and tested.
