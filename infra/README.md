# Infrastructure Deployment

## Prerequisites

1. Install Ansible role:
```bash
ansible-galaxy role install geerlingguy.docker -p roles
```

2. Set environment variables:
```bash
export TELEGRAM_BOT_TOKEN="your_bot_token_here"
```

## Deploy Lottie Converter

Deploy to production server:

```bash
cd infra
ansible-playbook -i hosts lottie.yml
```

This will:
- Install Docker and Docker Compose
- Build and start lottie-converter container
- Configure Traefik reverse proxy with HTTPS
- Service will be available at: https://lottie.trip2g.com

## Deploy Main App

```bash
cd infra
ansible-playbook -i hosts site.yml
```

## Custom Domain

To use a custom domain for lottie-converter, override the variable:

```bash
ansible-playbook -i hosts lottie.yml -e "lottie_domain=custom.domain.com"
```
