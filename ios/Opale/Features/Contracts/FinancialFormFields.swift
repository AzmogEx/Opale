import SwiftUI

struct FinancialFrequencyPicker: View {
    @Binding var value: String
    var allowOnce = false
    var body: some View {
        Picker("Fréquence", selection: $value) {
            if allowOnce { Text("Ponctuel").tag("once") }
            Text("Chaque mois").tag("monthly")
            Text("Chaque trimestre").tag("quarterly")
            Text("Chaque année").tag("yearly")
        }
    }
}
struct FinancialOptionalDate: View {
    let title: String
    @Binding var value: String
    var body: some View {
        Toggle(title, isOn: Binding(get: { !value.isEmpty }, set: { value = $0 ? Date.now.opaleDayString : "" }))
        if !value.isEmpty {
            DatePicker(title, selection: FinancialDate.binding($value), in: FinancialDate.allowedRange, displayedComponents: .date)
                .labelsHidden().frame(maxWidth: .infinity, alignment: .trailing)
        }
    }
}
enum FinancialDate {
    static var allowedRange: ClosedRange<Date> {
        Date.fromOpaleDay("1900-01-01")!...Calendar.opale.date(byAdding: .year, value: 10, to: .now)!
    }
    static var futureRange: ClosedRange<Date> {
        Calendar.opale.startOfDay(for: .now)...allowedRange.upperBound
    }
    static func binding(_ value: Binding<String>) -> Binding<Date> {
        Binding(get: { Date.fromOpaleDay(value.wrappedValue) ?? .now }, set: { value.wrappedValue = $0.opaleDayString })
    }
}
