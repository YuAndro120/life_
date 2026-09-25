import Testing
@testable import Life

@Suite struct SmokeTests {
    @Test func appModuleLoads() {
        #expect(Bool(true))
    }
}
