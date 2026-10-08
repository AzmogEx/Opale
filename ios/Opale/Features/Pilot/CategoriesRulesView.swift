import SwiftUI

struct CategoriesRulesView: View {
    @Environment(SessionStore.self) private var session
    @State private var categories: [Category] = []
    @State private var rules: [CategorizationRule] = []
    @State private var name = ""
    @State private var icon = "tag.fill"
    @State private var merchant = ""
    @State private var categoryID = ""
    @State private var error: String?
    @State private var busy = false
    @State private var editingCategory: Category?
    @State private var editingRule: CategorizationRule?
    var body: some View {
        Form {
            Section(editingCategory == nil ? "Créer une catégorie" : "Modifier la catégorie") {
                TextField("Nom", text: $name)
                Picker("Icône", selection: $icon) { ForEach(["tag.fill", "cart.fill", "house.fill", "car.fill", "fork.knife", "heart.fill", "gift.fill", "banknote.fill"], id: \.self) { Image(systemName: $0).tag($0) } }
                Button("Enregistrer") { Task { await saveCategory() } }.disabled(busy || name.trimmingCharacters(in: .whitespaces).isEmpty)
                if editingCategory != nil { Button("Annuler la modification") { editingCategory = nil; name = "" } }
            }
            Section("Catégories") {
                ForEach(categories) { c in Button { editingCategory = c; name = c.name; icon = c.icon } label: { Label(c.name, systemImage: c.icon) } }
                .onDelete { indices in Task { do { for i in indices { try await session.api.deleteCategory(id: categories[i].id) }; await load() } catch { self.error = error.localizedDescription } } }
            }
            Section(editingRule == nil ? "Créer une règle" : "Modifier la règle") {
                TextField("Clé du marchand", text: $merchant).textInputAutocapitalization(.never)
                Picker("Catégorie", selection: $categoryID) { Text("Choisir").tag(""); ForEach(categories) { Text($0.name).tag($0.id) } }
                Button("Enregistrer la règle") { Task { await saveRule() } }.disabled(busy || categoryID.isEmpty || merchant.isEmpty)
                if editingRule != nil { Button("Annuler la modification") { editingRule = nil; merchant = "" } }
            }
            Section("Règles apprises") {
                ForEach(rules) { r in Button { editingRule = r; merchant = r.merchantKey; categoryID = r.categoryID } label: { LabeledContent(r.merchantKey, value: categories.first { $0.id == r.categoryID }?.name ?? "Catégorie") } }
                .onDelete { indices in Task { do { for i in indices { try await session.api.deleteRule(id: rules[i].id) }; await load() } catch { self.error = error.localizedDescription } } }
            }
            ToolError(message: error)
        }.navigationTitle("Catégories & règles").task { await load() }
    }
    private func load() async { do { categories = try await session.api.listCategories(); rules = try await session.api.categorizationRules(); error = nil } catch { self.error = error.localizedDescription } }
    private func saveCategory() async { busy = true; defer { busy = false }; do { try await session.api.saveCategory(id: editingCategory?.id, name: name, icon: icon); editingCategory = nil; name = ""; session.changed(); await load() } catch { self.error = error.localizedDescription } }
    private func saveRule() async { busy = true; defer { busy = false }; do { try await session.api.saveRule(id: editingRule?.id, merchant: merchant, categoryID: categoryID); editingRule = nil; merchant = ""; await load() } catch { self.error = error.localizedDescription } }
}
