import SwiftUI

// Confidentialité & sécurité (ENF-004, EF-006) : verrouillage biométrique,
// journal d'accès, export des données.

/// Bouton de menu : activer/désactiver le verrouillage Face ID.
struct LockToggleButton: View {
    @Environment(AppLock.self) private var lock

    var body: some View {
        Button {
            lock.enabled.toggle()
        } label: {
            if lock.enabled {
                Label("Désactiver le verrouillage \(lock.biometryLabel)", systemImage: "lock.open")
            } else {
                Label("Verrouiller avec \(lock.biometryLabel)", systemImage: "faceid")
            }
        }
        .disabled(!lock.biometryAvailable)
    }
}

/// Journal d'accès (ENF-004) : « qui a consulté quoi ».
struct AccessLogSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var events: [AccessEvent] = []
    @State private var loaded = false

    var body: some View {
        NavigationStack {
            List {
                if loaded && events.isEmpty {
                    ContentUnavailableView(
                        "Journal vide",
                        systemImage: "list.bullet.rectangle",
                        description: Text("Les connexions, consultations de documents et exports apparaîtront ici.")
                    )
                }
                ForEach(events) { e in
                    HStack(alignment: .top, spacing: 10) {
                        Image(systemName: e.systemImage)
                            .foregroundStyle(e.isWarning ? OpaleTheme.loss : OpaleTheme.accent)
                            .frame(width: 26)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(e.label)
                                .font(.subheadline.weight(.medium))
                            HStack(spacing: 6) {
                                Text(e.at.formatted(.dateTime.day().month(.abbreviated).hour().minute()))
                                if !e.detail.isEmpty {
                                    Text("· \(e.detail)").lineLimit(1)
                                }
                            }
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        }
                    }
                }
            }
            .navigationTitle("Journal d'accès")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
            .task {
                events = (try? await session.api.accessLog()) ?? []
                loaded = true
            }
        }
    }
}

/// Partage de l'export complet (EF-006).
struct ExportShareSheet: View {
    let fileURL: URL

    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Image(systemName: "shippingbox.fill")
                    .font(.system(size: 44))
                    .foregroundStyle(OpaleTheme.iridescent)
                Text("Export prêt")
                    .font(.headline)
                Text("Toutes tes données dans un ZIP : patrimoine, mouvements, objectifs, contacts et les documents du coffre (déchiffrés). Garde-le en lieu sûr.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                ShareLink(item: fileURL) {
                    Label("Enregistrer / partager", systemImage: "square.and.arrow.up")
                        .padding(.horizontal, 8)
                }
                .buttonStyle(.glassProminent)
            }
            .padding()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
        }
    }
}
