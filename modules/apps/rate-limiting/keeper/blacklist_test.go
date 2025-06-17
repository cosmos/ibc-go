package keeper_test

import "slices"

func (s *KeeperTestSuite) TestDenomBlacklist() {
	allDenoms := []string{"denom1", "denom2", "denom3", "denom4"}
	denomsToBlacklist := []string{"denom1", "denom3"}

	// No denoms are currently blacklisted
	for _, denom := range allDenoms {
		isBlacklisted := s.chainA.GetSimApp().RateLimitKeeper.IsDenomBlacklisted(s.chainA.GetContext(), denom)
		s.Require().False(isBlacklisted, "%s should not be blacklisted yet", denom)
	}

	// Blacklist two denoms
	for _, denom := range denomsToBlacklist {
		s.chainA.GetSimApp().RateLimitKeeper.AddDenomToBlacklist(s.chainA.GetContext(), denom)
	}

	// Confirm half the list was blacklisted and the others were not
	for _, denom := range allDenoms {
		isBlacklisted := s.chainA.GetSimApp().RateLimitKeeper.IsDenomBlacklisted(s.chainA.GetContext(), denom)

		if slices.Contains(denomsToBlacklist, denom) {
			s.Require().True(isBlacklisted, "%s should have been blacklisted", denom)
			continue
		}
		s.Require().False(isBlacklisted, "%s should not have been blacklisted", denom)
	}
	actualBlacklistedDenoms := s.chainA.GetSimApp().RateLimitKeeper.GetAllBlacklistedDenoms(s.chainA.GetContext())
	s.Require().Len(actualBlacklistedDenoms, len(denomsToBlacklist), "number of blacklisted denoms")
	s.Require().ElementsMatch(denomsToBlacklist, actualBlacklistedDenoms, "list of blacklisted denoms")

	// Finally, remove denoms from blacklist and confirm they were removed
	for _, denom := range denomsToBlacklist {
		s.chainA.GetSimApp().RateLimitKeeper.RemoveDenomFromBlacklist(s.chainA.GetContext(), denom)
	}
	for _, denom := range allDenoms {
		isBlacklisted := s.chainA.GetSimApp().RateLimitKeeper.IsDenomBlacklisted(s.chainA.GetContext(), denom)

		if slices.Contains(denomsToBlacklist, denom) {
			s.Require().False(isBlacklisted, "%s should have been removed from the blacklist", denom)
			continue
		}
		s.Require().False(isBlacklisted, "%s should never have been blacklisted", denom)
	}
}
