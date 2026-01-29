# Maintainers

This document describes the governance and maintenance structure for Arcaflow MCP.

## Current Maintainers

| Name | GitHub | Role | Areas |
|------|--------|------|-------|
| Dustin Black | [@dustinblack](https://github.com/dustinblack) | Lead Maintainer | Architecture, Go server, Python engine |

**Becoming a maintainer**: Maintainers are added based on sustained contributions, technical expertise, and alignment with project values. If you're interested in becoming a maintainer, start by contributing consistently and engaging with the community.

---

## Roles and Responsibilities

### Lead Maintainer

**Responsibilities:**
- Set overall project direction and vision
- Make final decisions on controversial changes
- Ensure alignment with Arcalot community standards
- Manage releases and version planning
- Coordinate with Arcalot community
- Review and approve major architectural changes

### Maintainers

**Responsibilities:**
- Review and merge pull requests
- Triage issues and feature requests
- Maintain code quality standards
- Update documentation
- Participate in design discussions
- Support community members

**Expectations:**
- Timely responses to contributions as availability permits
- Adherence to project standards (see AGENTS.md)
- Constructive and respectful communication
- Regular participation in project activities

---

## Decision Making Process

### Consensus-Based Approach

Most decisions are made through consensus among maintainers:

1. **Proposal**: Anyone can propose changes via issues or discussions
2. **Discussion**: Maintainers and community members discuss trade-offs
3. **Consensus**: Maintainers agree on approach
4. **Implementation**: Code is submitted via pull request
5. **Review**: Maintainers review and approve

### When Consensus Fails

For controversial decisions:
- Extended discussion period (minimum 1 week)
- Lead maintainer makes final decision
- Decision is documented in ADR (Architecture Decision Record)
- Community is informed of rationale

### Fast-Track Decisions

Some changes can be fast-tracked:
- Bug fixes (no architectural impact)
- Documentation improvements
- Test additions
- Dependency updates (security)
- Minor refactoring

---

## Areas of Ownership

### Go MCP Server

**Owners**: Lead Maintainer  
**Components**: Protocol, transports, authentication, workflow tools  
**Code**: `server/`

**Major decisions require maintainer consensus:**
- MCP protocol changes
- New transport implementations
- Authentication/authorization changes
- Breaking API changes

### Python Analysis Engine

**Owners**: Lead Maintainer  
**Components**: Result analysis, optimization suggestions, metrics  
**Code**: `analysis/`

**Major decisions require maintainer consensus:**
- Analysis algorithm changes
- New analysis capabilities
- Database schema changes
- Breaking API changes

### Documentation

**Owners**: All maintainers  
**Components**: User docs, project docs, examples  
**Code**: `docs/`, README files

**Documentation changes are generally fast-tracked** unless they represent major directional changes.

### Infrastructure

**Owners**: Lead Maintainer  
**Components**: CI/CD, containers, deployment  
**Code**: `.github/`, `deploy/`, `scripts/`

**Major decisions require maintainer consensus:**
- CI/CD pipeline changes
- New deployment platforms
- Build system changes

---

## Contribution Review Process

### Pull Request Review

**Requirements for merge:**
- ✅ At least one maintainer approval
- ✅ All CI checks passing
- ✅ Tests included (minimum 85% coverage)
- ✅ Documentation updated
- ✅ No unresolved review comments

**Review timeline:**
- Initial review: As time permits
- Follow-up reviews: Promptly after contributor updates
- Final approval: When ready, no artificial delays

### Feature Requests

**Review process:**
1. **Triage**: Maintainers review new feature requests regularly
2. **Categorize**: Determine scope, priority, and alignment
3. **Respond**: Accept, defer, or decline with rationale
4. **Plan**: Accepted features added to roadmap
5. **Track**: Monitor progress and community interest

**Priority factors:**
- Community impact (how many users benefit?)
- Alignment with project vision
- Implementation complexity
- Maintenance burden
- Community contribution offers

### Bug Reports

**Triage process:**
1. **Verify**: Can it be reproduced?
2. **Assess severity**: Critical, high, medium, low
3. **Assign priority**: Immediate, next sprint, backlog
4. **Fix**: Implement fix or request community help
5. **Test**: Ensure regression test included

**Critical bugs** (security, data loss, crashes): Immediate attention  
**High priority bugs** (major functionality broken): Next sprint  
**Medium/low bugs**: Backlog, prioritized with features

---

## Release Management

### Release Process

**Owner**: Lead Maintainer  
**Process**: See [Release Process](docs/development/release-process.md)

**Version scheme**: Semantic versioning (MAJOR.MINOR.PATCH)

- **Major** (e.g., 1.0.0 → 2.0.0): Breaking changes, major features
- **Minor** (e.g., 0.1.0 → 0.2.0): New features, backwards compatible
- **Patch** (e.g., 0.1.0 → 0.1.1): Bug fixes, no new features

**Release cadence:**
- Major versions: Annually (post-v1.0)
- Minor versions: Quarterly
- Patch versions: As needed

### Release Responsibilities

- Maintainers ensure code quality and test coverage
- Lead maintainer coordinates release timing
- All maintainers review and approve release notes
- Community is notified of releases via GitHub and discussions

---

## Community Engagement

### Communication Channels

Maintainers monitor and engage in:
- **GitHub Issues**: Bug reports, feature requests
- **GitHub Pull Requests**: Code contributions
- **GitHub Discussions**: Community Q&A and ideas
- **Arcalot Round Table**: Broader Arcalot ecosystem discussions

### Maintainer Responsibilities

- Respond to issues and PRs in a timely manner
- Welcome new contributors and provide guidance
- Foster inclusive and respectful community
- Document decisions and rationale
- Keep roadmap and status docs updated

### Office Hours / Community Meetings

Currently informal. As the community grows, we may schedule:
- Monthly community calls
- Quarterly planning sessions
- Annual contributor summit

Announcements will be made in GitHub Discussions.

---

## Adding Maintainers

### Criteria for Maintainership

Potential maintainers should demonstrate:
1. **Sustained contributions**: Regular, high-quality contributions over 6+ months
2. **Technical expertise**: Deep understanding of relevant components
3. **Community engagement**: Helpful, constructive, and respectful
4. **Project alignment**: Shares project values and vision
5. **Commitment**: Available and willing to take on responsibilities

### Nomination Process

1. **Nomination**: Current maintainer nominates candidate
2. **Discussion**: Maintainers discuss candidate's qualifications (private)
3. **Consensus**: All current maintainers must agree
4. **Invitation**: Lead maintainer extends invitation
5. **Onboarding**: New maintainer is added and onboarded
6. **Announcement**: Community is notified

### Maintainer Emeritus

Maintainers who step back remain **Maintainers Emeritus**:
- Recognized for contributions
- Welcome to return to active role
- May be consulted on major decisions
- Listed in project history

---

## Alignment with Arcalot

Arcaflow MCP is part of the broader [Arcalot ecosystem](https://github.com/arcalot). We maintain alignment through:

### Governance Alignment

- Follow Arcalot [Code of Conduct](https://github.com/arcalot/.github/blob/main/CODE_OF_CONDUCT.md)
- Align with Arcalot project standards
- Coordinate on breaking changes that affect Arcaflow Engine
- Participate in Arcalot community discussions

### Technical Alignment

- Maintain compatibility with Arcaflow Engine
- Follow Arcaflow schema formats
- Use Arcalot-standard tooling (Go version, Python version, etc.)
- Contribute improvements back to Arcalot ecosystem

### License Alignment

- Apache 2.0 License (Arcalot standard)
- Compatible with other Arcalot projects
- No proprietary dependencies

---

## Conflict Resolution

### Disagreements Among Maintainers

1. **Discussion**: Extended discussion to understand perspectives
2. **Compromise**: Seek middle ground or alternative approaches
3. **Vote**: If needed, simple majority (lead maintainer has tie-breaking vote)
4. **Document**: Record decision and rationale in ADR
5. **Move forward**: Commit to supporting the decision

### Community Conflicts

- Follow [Code of Conduct](https://github.com/arcalot/.github/blob/main/CODE_OF_CONDUCT.md)
- Maintainers mediate disputes
- Focus on project goals, not personal issues
- Escalate to lead maintainer if needed

### Code of Conduct Violations

- Report to maintainers (privately if sensitive)
- Maintainers investigate and take appropriate action
- Actions range from warning to permanent ban
- Serious violations escalated to Arcalot community leadership

---

## Changes to Governance

This governance model may evolve as the project grows. Changes to this document:

1. Proposed via pull request
2. Discussed openly in GitHub
3. Require consensus among all maintainers
4. Announced to community before taking effect

---

## Questions?

- **General questions**: [GitHub Discussions](https://github.com/arcalot/arcalot-round-table/discussions)
- **Maintainer questions**: Open an issue or email maintainers directly
- **Private concerns**: Email lead maintainer directly

**Last Updated**: January 2026  
**Next Review**: Annually or as needed
