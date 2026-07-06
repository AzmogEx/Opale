import SwiftUI

/// Réglages — le centre de contrôle d'Opale : sécurité, confidentialité,
/// données (export / réinitialisation / suppression), serveur, devises.
struct SettingsView: View {
	@Environment(SessionStore.self) private var session
	@Environment(AppLock.self) private var lock
	@Environment(\.dismiss) private var dismiss

	@State private var status: AssistantStatus?
	@State private var bankStatus: BankStatus?

	// Feuilles
	@State private var showAccessLog = false
	@State private var showFXRates = false
	@State private var showBank = false
	@State private var exportedFileURL: URL?
	@State private var isExporting = false

	// Confirmations destructives : on exige le NOM EXACT du profil,
	// comme le backend (garde-fou à double détente).
	@State private var confirmReset = false
	@State private var confirmDelete = false
	@State private var typedName = ""
	@State private var feedback = ""

	var body: some View {
		NavigationStack {
			Form {
				profileSection
				securitySection
				privacySection
				dataSection
				serverSection
				aboutSection

				Section {
					Button("Se déconnecter", role: .destructive) {
						Task {
							await session.logout()
							dismiss()
						}
					}
				}
			}
			.navigationTitle("Réglages")
			.navigationBarTitleDisplayMode(.inline)
			.toolbar {
				ToolbarItem(placement: .cancellationAction) {
					Button("Fermer") { dismiss() }
				}
			}
			.task {
				status = try? await session.api.assistantStatus()
				bankStatus = try? await session.api.bankStatus()
			}
			.sheet(isPresented: $showAccessLog) { AccessLogSheet() }
			.sheet(isPresented: $showFXRates) { FXRatesSheet {} }
			.sheet(isPresented: $showBank) { BankSheet {} }
			.sheet(item: $exportedFileURL) { url in
				ExportShareSheet(fileURL: url)
					.presentationDetents([.medium])
			}
			.alert("Réinitialiser les données ?", isPresented: $confirmReset) {
				TextField("Tape « \(session.profileName) » pour confirmer", text: $typedName)
				Button("Tout effacer", role: .destructive) { Task { await resetData() } }
				Button("Annuler", role: .cancel) { typedName = "" }
			} message: {
				Text("Actifs, mouvements, objectifs, documents… tout le contenu du profil sera effacé. Le profil et son code restent. Irréversible — pense à exporter d'abord.")
			}
			.alert("Supprimer le profil ?", isPresented: $confirmDelete) {
				TextField("Tape « \(session.profileName) » pour confirmer", text: $typedName)
				Button("Supprimer définitivement", role: .destructive) { Task { await deleteProfile() } }
				Button("Annuler", role: .cancel) { typedName = "" }
			} message: {
				Text("Le profil et TOUTES ses données disparaissent définitivement.")
			}
		}
	}

	// MARK: - Sections

	private var profileSection: some View {
		Section("Profil") {
			LabeledContent("Connecté en tant que", value: session.profileName)
		}
	}

	private var securitySection: some View {
		Section {
			// Le binding déclenche Face ID IMMÉDIATEMENT (feedback visible) ;
			// si l'authentification échoue, le toggle revient tout seul.
			Toggle(isOn: Binding(
				get: { lock.enabled },
				set: { wanted in Task { await lock.setEnabled(wanted) } }
			)) {
				VStack(alignment: .leading, spacing: 2) {
					Text("Verrouillage \(lock.biometryLabel)")
					Text("Se verrouille en quittant l'app (retour à l'écran d'accueil)")
						.font(.caption)
						.foregroundStyle(.secondary)
				}
			}
			.disabled(!lock.biometryAvailable)

			if lock.enabled {
				Button {
					lock.lockNow()
					dismiss()
				} label: {
					Label("Verrouiller maintenant", systemImage: "lock.fill")
				}
			}

			Button {
				showAccessLog = true
			} label: {
				Label("Journal d'accès", systemImage: "list.bullet.rectangle")
			}
		} header: {
			Text("Sécurité")
		} footer: {
			if !lock.biometryAvailable {
				Text("Biométrie indisponible sur cet appareil (code non défini ?).")
			}
		}
	}

	private var privacySection: some View {
		Section("Confidentialité") {
			@Bindable var session = session
			Toggle(isOn: $session.discreetMode) {
				VStack(alignment: .leading, spacing: 2) {
					Text("Mode discret")
					Text("Floute tous les montants")
						.font(.caption)
						.foregroundStyle(.secondary)
				}
			}
			if let status {
				LabeledContent("IA homelab (N2)") {
					Label(status.homelabAvailable ? "En ligne" : "Hors ligne",
					      systemImage: status.homelabAvailable ? "checkmark.circle.fill" : "circle")
						.foregroundStyle(status.homelabAvailable ? OpaleTheme.gain : .secondary)
						.font(.subheadline)
				}
				LabeledContent("IA cloud (N3, anonymisé)") {
					Text(status.cloudConfigured ? "Configurée" : "Non configurée")
						.font(.subheadline)
						.foregroundStyle(status.cloudConfigured ? OpaleTheme.accent : .secondary)
				}
			}
		}
	}

	private var dataSection: some View {
		Section {
			Button {
				Task { await exportData() }
			} label: {
				Label(isExporting ? "Export en cours…" : "Exporter mes données (ZIP)",
				      systemImage: "square.and.arrow.up")
			}
			.disabled(isExporting)

			Button {
				showFXRates = true
			} label: {
				Label("Devises & taux", systemImage: "eurosign.arrow.circlepath")
			}

			Button {
				showBank = true
			} label: {
				Label {
					VStack(alignment: .leading, spacing: 2) {
						Text("Synchro bancaire")
						if let bankStatus {
							Text(bankStatus.configured
								? "\(bankStatus.links?.count ?? 0) banque(s) connectée(s)"
								: "Non configurée — import CSV")
								.font(.caption)
								.foregroundStyle(.secondary)
						}
					}
				} icon: {
					Image(systemName: "building.columns")
				}
			}

			Button(role: .destructive) {
				typedName = ""
				confirmReset = true
			} label: {
				Label("Réinitialiser toutes les données", systemImage: "arrow.counterclockwise")
			}

			Button(role: .destructive) {
				typedName = ""
				confirmDelete = true
			} label: {
				Label("Supprimer le profil", systemImage: "trash")
			}
		} header: {
			Text("Données")
		} footer: {
			if !feedback.isEmpty {
				Text(feedback).foregroundStyle(OpaleTheme.loss)
			}
		}
	}

	private var serverSection: some View {
		Section {
			LabeledContent("Serveur", value: session.baseURLString)
		} header: {
			Text("Serveur")
		} footer: {
			Text("L'adresse se change depuis l'écran de connexion (déconnecte-toi d'abord).")
		}
	}

	private var aboutSection: some View {
		Section("À propos") {
			LabeledContent("Version",
			               value: Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "—")
			LabeledContent("Philosophie", value: "Beau · Intelligent · Privé")
			Text("Le moteur calcule, l'IA explique. Tes données restent chez toi.")
				.font(.caption)
				.foregroundStyle(.secondary)
		}
	}

	// MARK: - Actions

	private func exportData() async {
		isExporting = true
		defer { isExporting = false }
		do {
			let data = try await session.api.exportData()
			let url = FileManager.default.temporaryDirectory
				.appendingPathComponent("opale-export.zip")
			try data.write(to: url)
			exportedFileURL = url
		} catch {
			feedback = "Export impossible : \(error.localizedDescription)"
		}
	}

	private func resetData() async {
		guard typedName == session.profileName else {
			feedback = "Confirmation incorrecte : tape exactement « \(session.profileName) »."
			return
		}
		do {
			try await session.api.resetData(confirmName: typedName)
			feedback = ""
			typedName = ""
			dismiss() // l'Accueil se recharge à vide
		} catch {
			feedback = error.localizedDescription
		}
	}

	private func deleteProfile() async {
		guard typedName == session.profileName else {
			feedback = "Confirmation incorrecte : tape exactement « \(session.profileName) »."
			return
		}
		do {
			try await session.api.deleteProfile(confirmName: typedName)
			await session.logout()
			dismiss()
		} catch {
			feedback = error.localizedDescription
		}
	}
}
