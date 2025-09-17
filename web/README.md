# Coco Web UI Development Guide

This guide covers everything you need to work on the Coco server's web interface.

## Development Environment Setup

### Prerequisites

- **Node.js**: >=18.12.0 (recommended 18.19.0+)
- **pnpm**: >=8.7.0 (recommended 8.14.0+)
- **Git**: For version control

### Quick Start

1. **Install dependencies** (use pnpm only):
   ```bash
   pnpm i
   ```

2. **Start the backend server**:
   - Download latest snapshot: https://release.infinilabs.com/coco/server/snapshot/
   - Installation docs: https://docs.infinilabs.com/coco-server/main/docs/getting-started/install/

3. **Configure backend connection**:
   Edit `.env.test`:
   ```
   VITE_SERVICE_BASE_URL=http://localhost:9000
   VITE_OTHER_SERVICE_BASE_URL={
     "demo": "http://localhost:9528",
     "/assets": "http://localhost:9000"
   }
   ```

4. **Start development server**:
   ```bash
   pnpm dev
   ```

## Development Commands

- `pnpm dev` - Start development server with hot reload
- `pnpm build` - Build production bundle
- `pnpm lint` - Run code linting
- `pnpm test` - Run test suite

## Project Structure

This is a pnpm monorepo. Key directories:
- `/apps` - Main application code
- `/packages` - Shared components and utilities
- `/widgets` - Reusable UI widgets

## Key Technologies

- **React** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Styling
- **pnpm workspaces** - Monorepo management

## Working with the Backend

The web UI communicates with the Coco server backend API. Default endpoints:
- Main API: `http://localhost:9000`
- Demo service: `http://localhost:9528`

Configuration is handled through environment variables in `.env.test`.
