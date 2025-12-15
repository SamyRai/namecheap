# DNS Provider Priority List

## Overview

This document lists DNS providers prioritized by market share, developer adoption, and API quality. This helps guide which providers to implement next for the zonekit project.

## Tier 1: Essential Providers (High Priority)

These are the most widely used providers and should be implemented first.

### 1. **Cloudflare** ⭐⭐⭐⭐⭐
- **Market Share**: ~19.3% of all websites (2025)
- **Why**:
  - Excellent API (REST, well-documented)
  - Free tier available
  - Very popular with developers
  - Fast DNS resolution
  - Great for sync/migration use cases
- **API**: REST API v4
- **Documentation**: Excellent
- **Difficulty**: Medium (REST API, straightforward)
- **Status**: Config example exists, needs implementation

### 2. **AWS Route 53** ⭐⭐⭐⭐⭐
- **Market Share**: Most popular in Americas, 16% of DNS traffic
- **Why**:
  - Industry standard for cloud deployments
  - Excellent API (AWS SDK)
  - Highly reliable
  - Common migration target
- **API**: AWS SDK / REST API
- **Documentation**: Excellent
- **Difficulty**: Medium (AWS SDK integration)
- **Status**: Not started

### 3. **GoDaddy** ⭐⭐⭐⭐
- **Market Share**: World's 5th largest web host, 62M+ domains
- **Why**:
  - Very popular domain registrar
  - Many users have domains there
  - Common migration source/target
- **API**: REST API
- **Documentation**: Good
- **Difficulty**: Medium
- **Status**: Not started

## Tier 2: Popular Providers (Medium Priority)

### 4. **DigitalOcean** ⭐⭐⭐⭐
- **Why**:
  - Popular with developers
  - Simple REST API
  - Good documentation
  - Often used for side projects
- **API**: REST API
- **Documentation**: Good
- **Difficulty**: Easy (simple REST API)
- **Status**: Not started

### 5. **Google Cloud DNS** ⭐⭐⭐⭐
- **Why**:
  - Part of GCP ecosystem
  - Good for Google Cloud users
  - Reliable infrastructure
- **API**: Google Cloud API
- **Documentation**: Good
- **Difficulty**: Medium (GCP authentication)
- **Status**: Not started

### 6. **Namecheap** ⭐⭐⭐⭐
- **Why**:
  - Popular domain registrar
  - Affordable pricing
  - Already implemented! ✅
- **API**: SOAP API (via SDK)
- **Documentation**: Good
- **Difficulty**: Medium (SOAP, special handling)
- **Status**: ✅ **IMPLEMENTED**

## Tier 3: Additional Providers (Lower Priority)

### 7. **Vercel** ⭐⭐⭐
- **Why**: Popular with Next.js developers
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

### 8. **Linode / Akamai** ⭐⭐⭐
- **Why**: Popular VPS provider with DNS
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

### 9. **Vultr** ⭐⭐⭐
- **Why**: Popular VPS provider
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

### 10. **Hetzner** ⭐⭐⭐
- **Why**: Popular European provider
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

### 11. **OVH** ⭐⭐⭐
- **Why**: Large European provider
- **API**: REST API
- **Difficulty**: Medium
- **Status**: Not started

### 12. **Name.com** ⭐⭐
- **Why**: Domain registrar
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

### 13. **Porkbun** ⭐⭐
- **Why**: Affordable domain registrar
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

### 14. **Dynu** ⭐⭐
- **Why**: Dynamic DNS provider
- **API**: REST API
- **Difficulty**: Easy
- **Status**: Not started

## Recommended Implementation Order

### Phase 1: Core Providers (Next 3)
1. ✅ **Namecheap** - DONE
2. 🔄 **Cloudflare** - HIGH PRIORITY (config exists)
3. 🔄 **AWS Route 53** - HIGH PRIORITY
4. 🔄 **GoDaddy** - HIGH PRIORITY

### Phase 2: Developer-Friendly (Next 2-3)
5. **DigitalOcean** - Easy API, popular
6. **Google Cloud DNS** - GCP ecosystem
7. **Vercel** - Next.js ecosystem

### Phase 3: Additional Coverage (As Needed)
8. **Linode/Akamai** - VPS users
9. **Vultr** - VPS users
10. **Others** - Based on user requests

## Selection Criteria

Providers were prioritized based on:

1. **Market Share**: How many domains/websites use them
2. **Developer Adoption**: Popular with developers using APIs
3. **API Quality**: Well-documented, stable APIs
4. **Migration Use Cases**: Common source/destination for DNS migrations
5. **Ease of Implementation**: Simpler APIs prioritized first

## API Complexity Notes

- **REST APIs**: Generally easier to implement (Cloudflare, DigitalOcean, GoDaddy)
- **SOAP APIs**: More complex (Namecheap - already done)
- **Cloud SDKs**: Medium complexity (AWS Route 53, Google Cloud DNS)
- **OAuth Required**: Adds complexity (some providers)

## Notes

- **Public DNS Resolvers** (Google 8.8.8.8, Cloudflare 1.1.1.1) are NOT included - these are DNS resolvers, not DNS hosting providers
- Focus is on providers that allow **DNS record management via API**
- All listed providers have public APIs for DNS management

## Resources

- Cloudflare API: https://developers.cloudflare.com/api/
- AWS Route 53 API: https://docs.aws.amazon.com/route53/
- GoDaddy API: https://developer.godaddy.com/
- DigitalOcean API: https://docs.digitalocean.com/reference/api/api-reference/

