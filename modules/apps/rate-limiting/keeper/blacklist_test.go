package keeper_test

// // Helper function to check if an element is in an array
// func isInArray(element string, arr []string) bool {
// 	for _, e := range arr {
// 		if e == element {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (s *KeeperTestSuite) TestDenomBlacklist() {
// 	allDenoms := []string{"denom1", "denom2", "denom3", "denom4"}
// 	denomsToBlacklist := []string{"denom1", "denom3"}

// 	// No denoms are currently blacklisted
// 	for _, denom := range allDenoms {
// 		isBlacklisted := s.App.RatelimitKeeper.IsDenomBlacklisted(s.Ctx, denom)
// 		s.Require().False(isBlacklisted, "%s should not be blacklisted yet", denom)
// 	}

// 	// Blacklist two denoms
// 	for _, denom := range denomsToBlacklist {
// 		s.App.RatelimitKeeper.AddDenomToBlacklist(s.Ctx, denom)
// 	}

// 	// Confirm half the list was blacklisted and the others were not
// 	for _, denom := range allDenoms {
// 		isBlacklisted := s.App.RatelimitKeeper.IsDenomBlacklisted(s.Ctx, denom)

// 		if isInArray(denom, denomsToBlacklist) {
// 			s.Require().True(isBlacklisted, "%s should have been blacklisted", denom)
// 		} else {
// 			s.Require().False(isBlacklisted, "%s should not have been blacklisted", denom)
// 		}
// 	}
// 	actualBlacklistedDenoms := s.App.RatelimitKeeper.GetAllBlacklistedDenoms(s.Ctx)
// 	s.Require().Len(actualBlacklistedDenoms, len(denomsToBlacklist), "number of blacklisted denoms")
// 	s.Require().ElementsMatch(denomsToBlacklist, actualBlacklistedDenoms, "list of blacklisted denoms")

// 	// Finally, remove denoms from blacklist and confirm they were removed
// 	for _, denom := range denomsToBlacklist {
// 		s.App.RatelimitKeeper.RemoveDenomFromBlacklist(s.Ctx, denom)
// 	}
// 	for _, denom := range allDenoms {
// 		isBlacklisted := s.App.RatelimitKeeper.IsDenomBlacklisted(s.Ctx, denom)

// 		if isInArray(denom, denomsToBlacklist) {
// 			s.Require().False(isBlacklisted, "%s should have been removed from the blacklist", denom)
// 		} else {
// 			s.Require().False(isBlacklisted, "%s should never have been blacklisted", denom)
// 		}
// 	}
// }