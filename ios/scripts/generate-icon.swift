// Icône d'Opale — générée par code (identité reproductible).
//
//   swift ios/scripts/generate-icon.swift ios/Opale/Assets.xcassets/AppIcon.appiconset/icon-1024.png
//
// Le concept : une opale — un anneau irisé (teal → pervenche → mauve) qui
// flotte sur un fond nuit profond, avec deux lueurs douces. Sobre, premium.

import CoreGraphics
import ImageIO
import UniformTypeIdentifiers
import Foundation

let size: CGFloat = 1024
let out = CommandLine.arguments.count > 1
	? CommandLine.arguments[1]
	: "icon-1024.png"

let colorSpace = CGColorSpace(name: CGColorSpace.sRGB)!
let ctx = CGContext(
	data: nil, width: Int(size), height: Int(size),
	bitsPerComponent: 8, bytesPerRow: 0, space: colorSpace,
	bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
)!

func rgba(_ r: CGFloat, _ g: CGFloat, _ b: CGFloat, _ a: CGFloat = 1) -> CGColor {
	CGColor(colorSpace: colorSpace, components: [r, g, b, a])!
}

// Palette Opale (alignée sur OpaleTheme).
let teal = rgba(0.35, 0.72, 0.71)
let periwinkle = rgba(0.49, 0.56, 0.84)
let mauve = rgba(0.71, 0.54, 0.79)

// ── Fond : nuit profonde, léger dégradé vertical ─────────────────────────────
let bg = CGGradient(
	colorsSpace: colorSpace,
	colors: [rgba(0.055, 0.07, 0.11), rgba(0.03, 0.04, 0.065)] as CFArray,
	locations: [0, 1]
)!
ctx.drawLinearGradient(bg, start: CGPoint(x: 0, y: size), end: CGPoint(x: 0, y: 0), options: [])

// ── Lueurs douces (teal en haut-gauche, mauve en bas-droite) ─────────────────
func glow(center: CGPoint, radius: CGFloat, color: CGColor, alpha: CGFloat) {
	let components = color.components!
	let soft = CGGradient(
		colorsSpace: colorSpace,
		colors: [
			rgba(components[0], components[1], components[2], alpha),
			rgba(components[0], components[1], components[2], 0),
		] as CFArray,
		locations: [0, 1]
	)!
	ctx.drawRadialGradient(soft, startCenter: center, startRadius: 0,
	                       endCenter: center, endRadius: radius, options: [])
}
glow(center: CGPoint(x: 250, y: 800), radius: 500, color: teal, alpha: 0.22)
glow(center: CGPoint(x: 800, y: 220), radius: 520, color: mauve, alpha: 0.18)

// ── L'anneau irisé (l'opale) ──────────────────────────────────────────────────
let center = CGPoint(x: size / 2, y: size / 2)
let ringRadius: CGFloat = 300
let ringWidth: CGFloat = 118

func circleRect(_ radius: CGFloat) -> CGRect {
	CGRect(x: center.x - radius, y: center.y - radius, width: radius * 2, height: radius * 2)
}

// Deux ellipses + règle pair-impair = un anneau fiable.
func ringPath() -> CGPath {
	let path = CGMutablePath()
	path.addEllipse(in: circleRect(ringRadius + ringWidth / 2))
	path.addEllipse(in: circleRect(ringRadius - ringWidth / 2))
	return path
}

// Halo derrière l'anneau.
glow(center: center, radius: ringRadius + 220, color: periwinkle, alpha: 0.16)

ctx.saveGState()
ctx.addPath(ringPath())
ctx.clip(using: .evenOdd)
let iridescent = CGGradient(
	colorsSpace: colorSpace,
	colors: [teal, periwinkle, mauve] as CFArray,
	locations: [0, 0.5, 1]
)!
ctx.drawLinearGradient(
	iridescent,
	start: CGPoint(x: center.x - ringRadius, y: center.y + ringRadius),
	end: CGPoint(x: center.x + ringRadius, y: center.y - ringRadius),
	options: [.drawsBeforeStartLocation, .drawsAfterEndLocation]
)
ctx.restoreGState()

// Reflet : un arc fin blanc en haut-gauche — le « verre » de l'opale.
ctx.saveGState()
ctx.setStrokeColor(rgba(1, 1, 1, 0.45))
ctx.setLineWidth(14)
ctx.setLineCap(.round)
ctx.addArc(center: center, radius: ringRadius + ringWidth / 2 - 20,
           startAngle: .pi * 0.55, endAngle: .pi * 0.95, clockwise: false)
ctx.strokePath()
ctx.restoreGState()

// Cœur : un point lumineux au centre — le patrimoine, condensé.
glow(center: center, radius: 130, color: teal, alpha: 0.5)
ctx.setFillColor(rgba(0.92, 0.98, 0.97))
ctx.fillEllipse(in: CGRect(x: center.x - 34, y: center.y - 34, width: 68, height: 68))

// ── Écriture du PNG ───────────────────────────────────────────────────────────
let image = ctx.makeImage()!
let url = URL(fileURLWithPath: out) as CFURL
let dest = CGImageDestinationCreateWithURL(url, UTType.png.identifier as CFString, 1, nil)!
CGImageDestinationAddImage(dest, image, nil)
CGImageDestinationFinalize(dest)
print("icône écrite : \(out)")
