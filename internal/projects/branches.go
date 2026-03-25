package projects

import (
	"path/filepath"
	"strings"
)

// NormalizePreferredBranchNames recorta, deduplica y deja como máximo 3 nombres.
func NormalizePreferredBranchNames(v []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range v {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

// NormalizePreferredBranchesMap normaliza claves a rutas absolutas limpias y los valores.
func NormalizePreferredBranchesMap(m map[string][]string) map[string][]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string][]string)
	for k, v := range m {
		abs, err := filepath.Abs(k)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		nv := NormalizePreferredBranchNames(v)
		if len(nv) == 0 {
			continue
		}
		out[abs] = nv
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ApplyPreferredFirst devuelve all reordenado: primero las que aparecen en preferred (en ese orden),
// luego el resto en el orden original de all. Solo cuenta coincidencias exactas de nombre de ref.
func ApplyPreferredFirst(all []string, preferred []string) []string {
	if len(preferred) == 0 {
		out := make([]string, len(all))
		copy(out, all)
		return out
	}
	seen := make(map[string]struct{})
	var head []string
	for _, p := range preferred {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		for _, a := range all {
			if a == p {
				if _, ok := seen[a]; !ok {
					head = append(head, a)
					seen[a] = struct{}{}
				}
				break
			}
		}
	}
	var rest []string
	for _, a := range all {
		if _, ok := seen[a]; !ok {
			rest = append(rest, a)
		}
	}
	return append(head, rest...)
}
