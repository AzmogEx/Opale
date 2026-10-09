import SwiftUI

/// Réglages — le centre de contrôle d'Opale : sécurité, confidentialité,
/// données (export / réinitialisation / suppression), serveur, devises.
struct SettingsView: View {
	@Environment(SessionStore.self) private var session
	@Environment(AppLock.self) private var lock
	@Environment(\.dismiss) private var dismiss

	@AppStorage("appearance.mode") private var appearanceMode = "system"
	@AppStorage("appearance.accent") private var accentColor = "opale"
	@State private var status: AssistantStatus?
	@State private var bankStatus: BankStatus?

	// Feuilles
	@State private var showAccessLog = false
	@State private var showFXRates = false
	@State private var showBank = false
	@State private var showFinancialSetup = false
    @State private var hasLegacyDraft = false
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
                Section("Outils") {
                    NavigationLink { AssistantSettingsView() } label: { Label("Choisir et connecter mon IA", systemImage: "server.rack") }
                    NavigationLink { ContractsView() } label: { Label("Contrats et abonnements", systemImage: "doc.text") }
                    NavigationLink { VariableIncomesView() } label: { Label("Revenus variables", systemImage: "waveform.path") }
                    NavigationLink { InvestmentExplorerView() } label: { Label("Explorer les investissements", systemImage: "globe.europe.africa") }
                    NavigationLink { PilotToolsView() } label: { Label("Pilote automatique", systemImage: "sparkles") }
                    NavigationLink { PushSettingsView() } label: { Label("Notifications", systemImage: "bell") }
                }
                Section("Apparence") {
                    Picker("Thème", selection: $appearanceMode) { Text("Système").tag("system"); Text("Clair").tag("light"); Text("Sombre").tag("dark") }
                    Picker("Accent", selection: $accentColor) { Text("Opale").tag("opale"); Text("Bleu").tag("blue"); Text("Violet").tag("purple"); Text("Orange").tag("orange") }
                }
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
			.opaleList()
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
                if let setup = try? await session.api.fetchFinancialSetup() {
                    hasLegacyDraft = setup.status == "draft" || setup.status == "skipped"
                }
			}
			.sheet(isPresented: $showAccessLog) { AccessLogSheet() }
			.sheet(isPresented: $showFXRates) { FXRatesSheet {} }
			.sheet(isPresented: $showBank) { BankSheet {} }
			.fullScreenCover(isPresented: $showFinancialSetup) { FinancialSetupView().id(session.profileKey) }
			.sheet(item: $exportedFileURL) { url in
				ExportShareSheet(fileURL: url)
					.presentationDetents([.medium])
                    .onDisappear { try? FileManager.default.removeItem(at: url) }
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
                Button { session.selectedTab = "home"; dismiss() } label: { Label("Mon parcours guidé", systemImage: "list.number") }
                if hasLegacyDraft {
                    DisclosureGroup("Brouillon de l’ancien formulaire") {
                        Text("Ce brouillon reste consultable. Vérifie les éléments déjà ajoutés dans le parcours avant de le finaliser, pour éviter les doublons.").font(.caption).foregroundStyle(.secondary)
                        Button("Reprendre l’ancien brouillon") { showFinancialSetup = true }
                    }
                }
                NavigationLink("Modifier le profil / confidentialité") { ProfileSettingsView() }
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
					Text("Au démarrage, en quittant l’app et après cinq minutes sans interaction")
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

	@State private var soundsOn = SoundPlayer.enabled

	private var privacySection: some View {
		Section("Confidentialité") {
			@Bindable var session = session
			Toggle(isOn: $soundsOn) {
				VStack(alignment: .leading, spacing: 2) {
					Text("Sons discrets")
					Text("Petits sons sur les moments clés (célébrations, envoi)")
						.font(.caption)
						.foregroundStyle(.secondary)
				}
			}
			.onChange(of: soundsOn) { _, new in SoundPlayer.enabled = new }
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
				LabeledContent("IA cloud (N3, minimisé)") {
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
			try data.write(to: url, options: [.atomic, .completeFileProtection])
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
            DiskCache.clear()
            WidgetBridge.clear()
            session.changed()
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
