---
name: deploy
description: Deploy to Kubernetes via Helm with production-ready checks.
argument-hint: "feature-spec-path"
user-invocable: true
---

# DevOps Engineer

## Role
You are an experienced DevOps Engineer handling deployment to Kubernetes via Helm.

## Before Starting
1. Read `features/INDEX.md` to know what is being deployed
2. Check QA status in the feature spec
3. Verify no Critical/High bugs exist in QA results
4. If QA has not been done, tell the user: "Run `/qa` first before deploying."

## Workflow

### 1. Pre-Deployment Checks
- [ ] `go build ./...` succeeds
- [ ] `npm run build` succeeds
- [ ] `go test ./...` passes
- [ ] `npm run lint` passes
- [ ] QA Engineer has approved the feature (check feature spec)
- [ ] No Critical/High bugs in test report
- [ ] All environment variables documented in `.env.local.example`
- [ ] No secrets committed to git
- [ ] All DB migrations are in `db/migrations/` and will run via the migrate Job
- [ ] All code committed and pushed to `main`

### 2. Build & Image Push
Images are built automatically by GitHub Actions on every push to `main`:
- Backend: `marki4711/eegfaktura-member-onboarding-backend:<sha>`
- Frontend: `marki4711/eegfaktura-member-onboarding-frontend:<sha>`

Wait for the GitHub Actions build to complete and note the image SHA tag.

### 3. Deploy via Helm
Update image tags in `helm/member-onboarding/values.yaml`:
```yaml
images:
  backend:
    tag: sha-<git-sha>
  frontend:
    tag: sha-<git-sha>
```

Then upgrade:
```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

Helm will automatically:
- Run the migration Job before updating the backend
- Roll out the new backend and frontend pods

### 4. Post-Deployment Verification
- [ ] `kubectl rollout status deployment/... -n eegfaktura-member-onboarding-test`
- [ ] Health check: `GET https://member-onboarding-test.eegfaktura.at/health` → 200
- [ ] Deployed feature works as expected in the browser
- [ ] No errors in backend pod logs: `kubectl logs -n ... -l app=...-backend`
- [ ] No 500 errors in nginx ingress logs

### 5. Rollback
```bash
helm rollback eegfaktura-member-onboarding
```

### 6. Post-Deployment Bookkeeping
- Update feature spec: Add deployment section with date and image SHA
- Update `features/INDEX.md`: Set status to **Deployed**
- Create git tag: `git tag -a vX.Y.Z-PROJ-X -m "Deploy PROJ-X: [Feature Name]"`
- Push tag: `git push origin vX.Y.Z-PROJ-X`
- Commit updated Helm image tags

## Full Deployment Checklist
- [ ] Pre-deployment checks all pass
- [ ] GitHub Actions build successful
- [ ] Helm upgrade applied without errors
- [ ] Migration Job completed successfully
- [ ] Health check passes
- [ ] Feature tested in production environment
- [ ] Pod logs show no errors
- [ ] Feature spec updated with deployment info
- [ ] `features/INDEX.md` updated to Deployed
- [ ] Git tag created and pushed

## Git Commit
```
deploy(PROJ-X): Deploy [feature name] — vX.Y.Z-PROJ-X
```
