import SwiftUI

/// Porte d'entrée : choisir un profil et se connecter par PIN (EF-001/EF-002),
/// ou créer le premier profil.
struct ProfileGateView: View {
    @Environment(SessionStore.self) private var session

    enum ViewState {
        case loading
        case error(String)
        case loaded([Profile])
    }

    @State private var viewState: ViewState = .loading
    @State private var selectedProfile: Profile?
    @State private var showCreate = false

    var body: some View {
        NavigationStack {
            Group {
                switch viewState {
                case .loading:
                    ProgressView()
                case .error(let message):
                    ContentUnavailableView {
                        Label("Serveur injoignable", systemImage: "bolt.horizontal.circle")
                    } description: {
                        Text(message)
                    } actions: {
                        Button("Réessayer") { Task { await load() } }
                            .buttonStyle(.borderedProminent)
                        serverField
                    }
                case .loaded(let profiles):
                    profileList(profiles)
                }
            }
            .navigationTitle("Opale")
            .task { await load() }
            .sheet(item: $selectedProfile) { profile in
                PINLoginSheet(profile: profile)
                    .presentationDetents([.height(320)])
            }
            .sheet(isPresented: $showCreate) {
                CreateProfileSheet()
                    .presentationDetents([.medium])
            }
        }
    }

    @ViewBuilder
    private func profileList(_ profiles: [Profile]) -> some View {
        List {
            if profiles.isEmpty {
                ContentUnavailableView(
                    "Bienvenue dans Opale",
                    systemImage: "circle.hexagongrid",
                    description: Text("Crée ton premier profil pour commencer à suivre ton patrimoine.")
                )
            } else {
                Section("Qui es-tu ?") {
                    ForEach(profiles) { profile in
                        Button {
                            selectedProfile = profile
                        } label: {
                            HStack {
                                Image(systemName: "person.crop.circle.fill")
                                    .font(.title2)
                                    .foregroundStyle(OpaleTheme.iridescent)
                                Text(profile.name)
                                    .font(.headline)
                                    .foregroundStyle(.primary)
                                Spacer()
                                Image(systemName: "chevron.right")
                                    .font(.caption)
                                    .foregroundStyle(.tertiary)
                            }
                        }
                    }
                }
            }

            Section {
                Button {
                    showCreate = true
                } label: {
                    Label("Nouveau profil", systemImage: "plus.circle.fill")
                }
            }
        }
        .refreshable { await load() }
    }

    private var serverField: some View {
        @Bindable var session = session
        return TextField("URL du serveur", text: $session.baseURLString)
            .textFieldStyle(.roundedBorder)
            .keyboardType(.URL)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .padding(.horizontal)
    }

    private func load() async {
        viewState = .loading
        do {
            viewState = .loaded(try await session.api.listProfiles())
        } catch {
            viewState = .error(error.localizedDescription)
        }
    }
}

/// Saisie du PIN pour un profil donné.
private struct PINLoginSheet: View {
    let profile: Profile
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var pin = ""
    @State private var errorMessage: String?
    @State private var isLoggingIn = false
    @FocusState private var pinFocused: Bool

    var body: some View {
        VStack(spacing: 24) {
            Image(systemName: "person.crop.circle.fill")
                .font(.system(size: 48))
                .foregroundStyle(OpaleTheme.iridescent)
            Text(profile.name)
                .font(.title2.bold())

            SecureField("Code PIN", text: $pin)
                .textFieldStyle(.roundedBorder)
                .keyboardType(.numberPad)
                .multilineTextAlignment(.center)
                .frame(width: 160)
                .focused($pinFocused)
                .onSubmit { Task { await login() } }

            if let errorMessage {
                Text(errorMessage)
                    .font(.footnote)
                    .foregroundStyle(OpaleTheme.loss)
            }

            Button {
                Task { await login() }
            } label: {
                if isLoggingIn {
                    ProgressView().frame(maxWidth: .infinity)
                } else {
                    Text("Déverrouiller").frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(pin.isEmpty || isLoggingIn)
        }
        .padding(24)
        .onAppear { pinFocused = true }
    }

    private func login() async {
        isLoggingIn = true
        defer { isLoggingIn = false }
        do {
            try await session.login(profileID: profile.id, pin: pin)
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
            pin = ""
        }
    }
}

/// Création d'un profil (nom + PIN) puis connexion immédiate.
private struct CreateProfileSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var pin = ""
    @State private var pinConfirm = ""
    @State private var errorMessage: String?
    @State private var isCreating = false

    private var isValid: Bool {
        !name.trimmingCharacters(in: .whitespaces).isEmpty
            && pin.count >= 4
            && pin == pinConfirm
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Profil") {
                    TextField("Prénom", text: $name)
                }
                Section("Code PIN (4 chiffres min.)") {
                    SecureField("PIN", text: $pin)
                        .keyboardType(.numberPad)
                    SecureField("Confirme le PIN", text: $pinConfirm)
                        .keyboardType(.numberPad)
                }
                if let errorMessage {
                    Text(errorMessage)
                        .foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Nouveau profil")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Créer") { Task { await create() } }
                        .disabled(!isValid || isCreating)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
        }
    }

    private func create() async {
        isCreating = true
        defer { isCreating = false }
        do {
            try await session.createProfileAndLogin(
                name: name.trimmingCharacters(in: .whitespaces),
                pin: pin
            )
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
