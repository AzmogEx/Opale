#!/usr/bin/env python3
"""Check explicit feature coverage and regenerate the HTTP reference (stdlib only)."""
import argparse
import json
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]


def routes():
    """Read the nested Chi router; ignore braces inside route parameters/comments."""
    result, stack, depth = [], [], 0
    for number, raw in enumerate((ROOT / 'backend/internal/api/server.go').read_text().splitlines(), 1):
        line = raw.split('//', 1)[0]
        nested = re.search(r'r\.Route\("([^"]+)"', line)
        if nested:
            stack.append((depth, nested[1]))
        match = re.search(r'r\.(Get|Post|Put|Patch|Delete|Head|Options)\("([^"]+)", s\.(\w+)\)', line)
        if match:
            path = (''.join(prefix for _, prefix in stack) + match[2]).rstrip('/') or '/'
            result.append((match[1].upper() + ' ' + path, match[3], number))
        code = re.sub(r'"(?:[^"\\]|\\.)*"', '""', line)
        depth += code.count('{') - code.count('}')
        while stack and depth <= stack[-1][0]:
            stack.pop()
    return result


def screens():
    return set(str(p.relative_to(ROOT)) for pattern in (
        'ios/Opale/Features/**/*.swift', 'web/src/lib/components/*.svelte',
        'web/src/routes/**/*.svelte'
    ) for p in ROOT.glob(pattern))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write', action='store_true', help='Regenerate docs/API.md after coverage checks')
    args = parser.parse_args()
    catalog = json.loads((ROOT / 'docs/catalogue.json').read_text())
    documented_routes, documented_sources, problems = {}, set(), []
    for feature in catalog:
        doc = ROOT / feature['doc']
        if not doc.is_file():
            problems.append('Missing feature document: ' + feature['doc'])
            continue
        content = doc.read_text()
        for heading in ('## Utilisation', '## Données et comportement', '## Limites et configuration', '## Vérification'):
            if heading not in content:
                problems.append(f'{feature["doc"]}: missing {heading}')
        for route in feature['routes']:
            if route in documented_routes:
                problems.append('Route documented twice: ' + route)
            documented_routes[route] = feature
        for source in feature['sources']:
            documented_sources.add(source)
            if not (ROOT / source).is_file():
                problems.append('Missing source: ' + source)
    actual = routes()
    actual_keys = {route for route, _, _ in actual}
    problems += ['Undocumented route: ' + route for route in sorted(actual_keys - documented_routes.keys())]
    problems += ['Obsolete route: ' + route for route in sorted(documented_routes.keys() - actual_keys)]
    problems += ['Undocumented screen: ' + source for source in sorted(screens() - documented_sources)]
    # Local Markdown links must resolve; web URLs and anchors are outside this check.
    for doc in (ROOT / 'docs').rglob('*.md'):
        for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)', doc.read_text()):
            if re.match(r'^[a-zA-Z][a-zA-Z0-9+.-]*:', target) or target.startswith('#'):
                continue
            target = target.split('#', 1)[0]
            if target and not (doc.parent / target).exists():
                problems.append(f'{doc.relative_to(ROOT)}: broken link {target}')
    if problems:
        print('\n'.join(problems), file=sys.stderr)
        return 1
    lines = [
        '# Référence des routes HTTP', '',
        '> Générée par `python3 scripts/check_docs.py --write` depuis le routeur Chi et le catalogue documentaire. Ne pas éditer la table à la main.', '',
        'Backend public imposé : `https://opale.vaycode.com`. Les chemins de collection sont normalisés sans slash final dans cette table ; le routeur peut enregistrer leur variante `/`.', '',
        'Les routes `/v1` sont privées avec `Authorization: Bearer <jeton>`, sauf la liste/création/démo de profils et la connexion. Les erreurs sont des réponses HTTP explicites ; un 401 invalide une session authentifiée, un 403 ou une panne réseau ne signifie pas une expiration. Les écritures à révision refusent les conflits (409). Les montants `*_cents` sont des entiers en unités mineures de la devise, jamais des nombres décimaux JSON.', '',
        '`/healthz` vérifie la vie du processus ; `/readyz` vérifie sa disponibilité et PostgreSQL. `/metrics` ne demande pas de session au routeur mais reste bloqué par nginx public et réservé au réseau API privé.', '',
        'Chaque ligne renvoie à la fiche fonctionnelle et au handler pour le contrat exact de paramètres, validation et réponse. Les conventions transverses sont dans [CONVENTIONS-FINANCIERES.md](CONVENTIONS-FINANCIERES.md) et [DATA-MODEL.md](DATA-MODEL.md).', '',
        '| Méthode et chemin | Fonctionnalité | Handler |', '|---|---|---|'
    ]
    handlers = {}
    for source in sorted((ROOT / 'backend/internal/api').glob('*.go')):
        if source.name.endswith('_test.go'):
            continue
        for match in re.finditer(r'func \(s \*Server\) (handle\w+)\(', source.read_text()):
            handlers[match[1]] = str(source.relative_to(ROOT))
    for route, handler, _ in actual:
        feature = documented_routes[route]
        path = feature['doc'].removeprefix('docs/')
        source = handlers.get(handler, 'backend/internal/api/server.go')
        lines.append(f'| `{route}` | [{feature["title"]}]({path}) | [{handler}](../{source}) |')
    lines += ['', f'{len(actual)} routes, {len(catalog)} fiches, {len(screens())} écrans/composants couverts explicitement.', '']
    generated = '\n'.join(lines)
    api = ROOT / 'docs/API.md'
    if args.write:
        api.write_text(generated)
    elif not api.is_file() or api.read_text() != generated:
        print('docs/API.md is stale; run python3 scripts/check_docs.py --write', file=sys.stderr)
        return 1
    print(f'Documentation OK: {len(catalog)} fiches, {len(actual)} routes, {len(screens())} écrans/composants; liens locaux valides.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
