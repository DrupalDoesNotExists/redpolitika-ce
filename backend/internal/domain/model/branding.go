package model

import (
	"fmt"
	"strings"
)

// NavbarLink — ссылка в верхней панели навигации.
type NavbarLink struct {
	label string
	href  string
}

// NewNavbarLink creates a validated navbar link.
func NewNavbarLink(label, href string) (NavbarLink, error) {
	if label == "" {
		return NavbarLink{}, &DomainError{Op: "NewNavbarLink", Message: "label must not be empty"}
	}
	if href == "" {
		return NavbarLink{}, &DomainError{Op: "NewNavbarLink", Message: "href must not be empty"}
	}
	if err := validateBrandingHref(href); err != nil {
		return NavbarLink{}, err
	}
	return NavbarLink{label: label, href: href}, nil
}

func (l NavbarLink) Label() string { return l.label }
func (l NavbarLink) Href() string  { return l.href }

// Branding — value object white-label/co-branding (A36).
type Branding struct {
	productName string
	logoText    string
	logoHref    string
	partnerLogo string
	theme       map[string]string
	footerText  string
	footerHref  string
	navbarLinks []NavbarLink
}

// NewBranding creates Branding with validation.
// LogoText and LogoHref are required; LogoHref must be "/" or start with "/" or "http".
func NewBranding(
	productName, logoText, logoHref, partnerLogo string,
	theme map[string]string,
	footerText, footerHref string,
	navbarLinks []NavbarLink,
) (Branding, error) {
	if logoText == "" {
		return Branding{}, &DomainError{Op: "NewBranding", Message: "logo text must not be empty"}
	}
	if logoHref == "" {
		return Branding{}, &DomainError{Op: "NewBranding", Message: "logo href must not be empty"}
	}
	if err := validateBrandingHref(logoHref); err != nil {
		return Branding{}, err
	}
	if footerHref != "" {
		if err := validateBrandingHref(footerHref); err != nil {
			return Branding{}, err
		}
	}
	for i, link := range navbarLinks {
		if link.label == "" {
			return Branding{}, &DomainError{
				Op:      "NewBranding",
				Message: "navbar link label must not be empty",
			}
		}
		if link.href == "" {
			return Branding{}, &DomainError{
				Op:      "NewBranding",
				Message: "navbar link href must not be empty",
			}
		}
		if err := validateBrandingHref(link.href); err != nil {
			return Branding{}, &DomainError{
				Op:      "NewBranding",
				Message: fmt.Sprintf("navbar link %d: %s", i, err.Error()),
			}
		}
	}

	return Branding{
		productName: productName,
		logoText:    logoText,
		logoHref:    logoHref,
		partnerLogo: partnerLogo,
		theme:       cloneStringMap(theme),
		footerText:  footerText,
		footerHref:  footerHref,
		navbarLinks: cloneNavbarLinks(navbarLinks),
	}, nil
}

// DefaultBranding returns CE defaults (A36).
func DefaultBranding() Branding {
	b, err := NewBranding(
		"Редполитика",
		"Редполитикаᵝ",
		"/",
		"",
		nil,
		"Работает на движке Редполитикаᵝ",
		"https://github.com/drupaldoesnotexists/redpolitika-ce",
		nil,
	)
	if err != nil {
		panic("DefaultBranding: " + err.Error())
	}
	return b
}

// Merge returns a copy with non-empty fields from other overriding this branding.
func (b Branding) Merge(other Branding) Branding {
	out := b

	if other.productName != "" {
		out.productName = other.productName
	}
	if other.logoText != "" {
		out.logoText = other.logoText
	}
	if other.logoHref != "" {
		out.logoHref = other.logoHref
	}
	if other.partnerLogo != "" {
		out.partnerLogo = other.partnerLogo
	}
	if other.footerText != "" {
		out.footerText = other.footerText
	}
	if other.footerHref != "" {
		out.footerHref = other.footerHref
	}
	if len(other.theme) > 0 {
		if out.theme == nil {
			out.theme = make(map[string]string, len(other.theme))
		}
		for k, v := range other.theme {
			if v != "" {
				out.theme[k] = v
			}
		}
	}
	if len(other.navbarLinks) > 0 {
		out.navbarLinks = cloneNavbarLinks(other.navbarLinks)
	}

	return out
}

func (b Branding) ProductName() string        { return b.productName }
func (b Branding) LogoText() string           { return b.logoText }
func (b Branding) LogoHref() string           { return b.logoHref }
func (b Branding) PartnerLogo() string        { return b.partnerLogo }
func (b Branding) FooterText() string         { return b.footerText }
func (b Branding) FooterHref() string         { return b.footerHref }
func (b Branding) Theme() map[string]string   { return cloneStringMap(b.theme) }
func (b Branding) NavbarLinks() []NavbarLink  { return cloneNavbarLinks(b.navbarLinks) }

func validateBrandingHref(href string) error {
	if href == "/" {
		return nil
	}
	if strings.HasPrefix(href, "/") {
		return nil
	}
	if strings.HasPrefix(href, "http") {
		return nil
	}
	return &DomainError{
		Op:      "validateBrandingHref",
		Message: "href must be \"/\" or start with \"/\" or \"http\"",
	}
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneNavbarLinks(links []NavbarLink) []NavbarLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]NavbarLink, len(links))
	copy(out, links)
	return out
}
