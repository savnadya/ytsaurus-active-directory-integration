package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	testclock "k8s.io/utils/clock/testing"

	"go.ytsaurus.tech/yt/go/yt"
)

const (
	ytDevToken = "password"
)

type testCase struct {
	name      string
	appConfig *AppConfig
	testTime  time.Time

	azureUsersSetUp []SourceUser
	ytUsersSetUp    []YtsaurusUser
	ytUsersExpected []YtsaurusUser

	azureGroupsSetUp []SourceGroupWithMembers
	ytGroupsSetUp    []YtsaurusGroupWithMembers
	ytGroupsExpected []YtsaurusGroupWithMembers
}

var (
	testTimeStr     = "2023-10-20T12:00:00Z"
	initialTestTime = parseAppTime(testTimeStr)

	aliceAzure = AzureUser{
		PrincipalName: "alice@acme.com",
		AzureID:       "fake-az-id-alice",
		Email:         "alice@acme.com",
		FirstName:     "Alice",
		LastName:      "Henderson",
		DisplayName:   "Henderson, Alice (ACME)",
	}
	bobAzure = AzureUser{
		PrincipalName: "Bob@acme.com",
		AzureID:       "fake-az-id-bob",
		Email:         "Bob@acme.com",
		FirstName:     "Bob",
		LastName:      "Sanders",
		DisplayName:   "Sanders, Bob (ACME)",
	}
	carolAzure = AzureUser{
		PrincipalName: "carol@acme.com",
		AzureID:       "fake-az-id-carol",
		Email:         "carol@acme.com",
		FirstName:     "Carol",
		LastName:      "Sanders",
		DisplayName:   "Sanders, Carol (ACME)",
	}
	aliceAzureChangedLastName = AzureUser{
		PrincipalName: aliceAzure.PrincipalName,
		AzureID:       aliceAzure.AzureID,
		Email:         aliceAzure.Email,
		FirstName:     aliceAzure.FirstName,
		LastName:      "Smith",
		DisplayName:   aliceAzure.DisplayName,
	}
	bobAzureChangedEmail = AzureUser{
		PrincipalName: "bobby@example.com",
		AzureID:       bobAzure.AzureID,
		Email:         "bobby@example.com",
		FirstName:     bobAzure.FirstName,
		LastName:      bobAzure.LastName,
		DisplayName:   bobAzure.DisplayName,
	}
	devsAzureGroup = AzureGroup{
		Identity:    "acme.devs|all",
		AzureID:     "fake-az-acme.devs",
		DisplayName: "acme.devs|all",
	}
	hqAzureGroup = AzureGroup{
		Identity:    "acme.hq",
		AzureID:     "fake-az-acme.hq",
		DisplayName: "acme.hq",
	}
	devsAzureGroupChangedDisplayName = AzureGroup{
		Identity:    "acme.developers|all",
		AzureID:     devsAzureGroup.AzureID,
		DisplayName: "acme.developers|all",
	}
	hqAzureGroupChangedBackwardCompatible = AzureGroup{
		Identity:    "acme.hq|all",
		AzureID:     hqAzureGroup.AzureID,
		DisplayName: "acme.hq|all",
	}

	aliceYtsaurus = YtsaurusUser{
		Username: "alice",
		SourceRaw: map[string]any{
			"id":             aliceAzure.AzureID,
			"principal_name": aliceAzure.PrincipalName,
			"email":          aliceAzure.Email,
			"first_name":     aliceAzure.FirstName,
			"last_name":      aliceAzure.LastName,
			"display_name":   aliceAzure.DisplayName,
		},
	}
	bobYtsaurus = YtsaurusUser{
		Username: "bob",
		SourceRaw: map[string]any{
			"id":             bobAzure.AzureID,
			"principal_name": bobAzure.PrincipalName,
			"email":          bobAzure.Email,
			"first_name":     bobAzure.FirstName,
			"last_name":      bobAzure.LastName,
			"display_name":   bobAzure.DisplayName,
		},
	}
	carolYtsaurus = YtsaurusUser{
		Username: "carol",
		SourceRaw: map[string]any{
			"id":             carolAzure.AzureID,
			"principal_name": carolAzure.PrincipalName,
			"email":          carolAzure.Email,
			"first_name":     carolAzure.FirstName,
			"last_name":      carolAzure.LastName,
			"display_name":   carolAzure.DisplayName,
		},
	}
	aliceYtsaurusChangedLastName = YtsaurusUser{
		Username: aliceYtsaurus.Username,
		SourceRaw: map[string]any{
			"id":             aliceYtsaurus.SourceRaw["id"],
			"principal_name": aliceYtsaurus.SourceRaw["principal_name"],
			"email":          aliceYtsaurus.SourceRaw["email"],
			"first_name":     aliceYtsaurus.SourceRaw["first_name"],
			"last_name":      aliceAzureChangedLastName.LastName,
			"display_name":   aliceYtsaurus.SourceRaw["display_name"],
		},
	}
	bobYtsaurusChangedEmail = YtsaurusUser{
		Username: "bobby:example.com",
		SourceRaw: map[string]any{
			"id":             bobYtsaurus.SourceRaw["id"],
			"principal_name": bobAzureChangedEmail.PrincipalName,
			"email":          bobAzureChangedEmail.Email,
			"first_name":     bobYtsaurus.SourceRaw["first_name"],
			"last_name":      bobYtsaurus.SourceRaw["last_name"],
			"display_name":   bobYtsaurus.SourceRaw["display_name"],
		},
	}
	bobYtsaurusBanned = YtsaurusUser{
		Username: bobYtsaurus.Username,
		SourceRaw: map[string]any{
			"id":             bobYtsaurus.SourceRaw["id"],
			"principal_name": bobYtsaurus.SourceRaw["principal_name"],
			"email":          bobYtsaurus.SourceRaw["email"],
			"first_name":     bobYtsaurus.SourceRaw["first_name"],
			"last_name":      bobYtsaurus.SourceRaw["last_name"],
			"display_name":   bobYtsaurus.SourceRaw["display_name"],
		},
		BannedSince: initialTestTime,
	}
	carolYtsaurusBanned = YtsaurusUser{
		Username: carolYtsaurus.Username,
		SourceRaw: map[string]any{
			"id":             carolYtsaurus.SourceRaw["id"],
			"principal_name": carolYtsaurus.SourceRaw["principal_name"],
			"email":          carolYtsaurus.SourceRaw["email"],
			"first_name":     carolYtsaurus.SourceRaw["first_name"],
			"last_name":      carolYtsaurus.SourceRaw["last_name"],
			"display_name":   carolYtsaurus.SourceRaw["display_name"],
		},
		BannedSince: initialTestTime.Add(40 * time.Hour),
	}
	devsYtsaurusGroup = YtsaurusGroup{
		Name: "acme.devs",
		SourceRaw: map[string]any{
			"id":           devsAzureGroup.AzureID,
			"display_name": "acme.devs|all",
			"identity":     "acme.devs|all",
		},
	}
	qaYtsaurusGroup = YtsaurusGroup{
		Name: "acme.qa",
		SourceRaw: map[string]any{
			"id":           "fake-az-acme.qa",
			"display_name": "acme.qa|all",
			"identity":     "acme.qa",
		},
	}
	hqYtsaurusGroup = YtsaurusGroup{
		Name: "acme.hq",
		SourceRaw: map[string]any{
			"id":           hqAzureGroup.AzureID,
			"display_name": "acme.hq",
			"identity":     "acme.hq",
		},
	}
	devsYtsaurusGroupChangedDisplayName = YtsaurusGroup{
		Name: "acme.developers",
		SourceRaw: map[string]any{
			"id":           devsAzureGroup.AzureID,
			"display_name": "acme.developers|all",
			"identity":     "acme.developers|all",
		},
	}
	hqYtsaurusGroupChangedBackwardCompatible = YtsaurusGroup{
		Name: "acme.hq",
		SourceRaw: map[string]any{
			"id":           hqAzureGroup.AzureID,
			"display_name": "acme.hq|all",
			"identity":     "acme.hq|all",
		},
	}

	defaultUsernameReplacements = []ReplacementPair{
		{"@acme.com", ""},
		{"@", ":"},
	}
	defaultGroupnameReplacements = []ReplacementPair{
		{"|all", ""},
	}
	defaultAppConfig = &AppConfig{
		UsernameReplacements:  defaultUsernameReplacements,
		GroupnameReplacements: defaultGroupnameReplacements,
	}

	// we test several things in each test case, because of long wait for local ytsaurus
	// container start.
	testCases = []testCase{
		{
			name: "a-skip-b-create-c-remove",
			azureUsersSetUp: []SourceUser{
				aliceAzure,
				bobAzure,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				carolYtsaurus,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
			},
		},
		{
			name: "bob-is-banned",
			appConfig: &AppConfig{
				UsernameReplacements:    defaultUsernameReplacements,
				GroupnameReplacements:   defaultGroupnameReplacements,
				BanBeforeRemoveDuration: 24 * time.Hour,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
			},
			azureUsersSetUp: []SourceUser{
				aliceAzure,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurusBanned,
			},
		},
		{
			name: "bob-was-banned-now-deleted-carol-was-banned-now-back",
			// Bob was banned at initialTestTime,
			// 2 days have passed (more than setting allows) —> he should be removed.
			// Carol was banned 8 hours ago and has been found in Azure -> she should be restored.
			testTime: initialTestTime.Add(48 * time.Hour),
			appConfig: &AppConfig{
				UsernameReplacements:    defaultUsernameReplacements,
				GroupnameReplacements:   defaultGroupnameReplacements,
				BanBeforeRemoveDuration: 24 * time.Hour,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurusBanned,
				carolYtsaurusBanned,
			},
			azureUsersSetUp: []SourceUser{
				aliceAzure,
				carolAzure,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				carolYtsaurus,
			},
		},
		{
			name: "remove-limit-users-3",
			appConfig: &AppConfig{
				UsernameReplacements:  defaultUsernameReplacements,
				GroupnameReplacements: defaultGroupnameReplacements,
				RemoveLimit:           3,
			},
			azureUsersSetUp: []SourceUser{},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			// no one is deleted: limitation works
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
		},
		{
			name: "remove-limit-groups-3",
			appConfig: &AppConfig{
				UsernameReplacements:  defaultUsernameReplacements,
				GroupnameReplacements: defaultGroupnameReplacements,
				RemoveLimit:           3,
			},
			azureGroupsSetUp: []SourceGroupWithMembers{},
			ytGroupsSetUp: []YtsaurusGroupWithMembers{
				NewEmptyYtsaurusGroupWithMembers(devsYtsaurusGroup),
				NewEmptyYtsaurusGroupWithMembers(qaYtsaurusGroup),
				NewEmptyYtsaurusGroupWithMembers(hqYtsaurusGroup),
			},
			// no group is deleted: limitation works
			ytGroupsExpected: []YtsaurusGroupWithMembers{
				NewEmptyYtsaurusGroupWithMembers(devsYtsaurusGroup),
				NewEmptyYtsaurusGroupWithMembers(qaYtsaurusGroup),
				NewEmptyYtsaurusGroupWithMembers(hqYtsaurusGroup),
			},
		},
		{
			name: "a-changed-name-b-changed-email",
			azureUsersSetUp: []SourceUser{
				aliceAzureChangedLastName,
				bobAzureChangedEmail,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurusChangedLastName,
				bobYtsaurusChangedEmail,
			},
		},
		{
			name: "skip-create-remove-group-no-members-change-correct-name-replace",
			azureUsersSetUp: []SourceUser{
				aliceAzure,
				bobAzure,
				carolAzure,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			ytGroupsSetUp: []YtsaurusGroupWithMembers{
				{
					YtsaurusGroup: devsYtsaurusGroup,
					Members:       NewStringSetFromItems(aliceYtsaurus.Username),
				},
				{
					YtsaurusGroup: qaYtsaurusGroup,
					Members:       NewStringSetFromItems(bobYtsaurus.Username),
				},
			},
			azureGroupsSetUp: []SourceGroupWithMembers{
				{
					SourceGroup: devsAzureGroup,
					Members:     NewStringSetFromItems(aliceAzure.AzureID),
				},
				{
					SourceGroup: hqAzureGroup,
					Members:     NewStringSetFromItems(carolAzure.AzureID),
				},
			},
			ytGroupsExpected: []YtsaurusGroupWithMembers{
				{
					YtsaurusGroup: devsYtsaurusGroup,
					Members:       NewStringSetFromItems(aliceYtsaurus.Username),
				},
				{
					YtsaurusGroup: hqYtsaurusGroup,
					Members:       NewStringSetFromItems(carolYtsaurus.Username),
				},
			},
		},
		{
			name: "memberships-add-remove",
			azureUsersSetUp: []SourceUser{
				aliceAzure,
				bobAzure,
				carolAzure,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			ytGroupsSetUp: []YtsaurusGroupWithMembers{
				{
					YtsaurusGroup: devsYtsaurusGroup,
					Members: NewStringSetFromItems(
						aliceYtsaurus.Username,
						bobYtsaurus.Username,
					),
				},
			},
			azureGroupsSetUp: []SourceGroupWithMembers{
				{
					SourceGroup: devsAzureGroup,
					Members: NewStringSetFromItems(
						aliceAzure.AzureID,
						carolAzure.AzureID,
					),
				},
			},
			ytGroupsExpected: []YtsaurusGroupWithMembers{
				{
					YtsaurusGroup: devsYtsaurusGroup,
					Members: NewStringSetFromItems(
						aliceYtsaurus.Username,
						carolYtsaurus.Username,
					),
				},
			},
		},
		{
			name: "display-name-changes",
			azureUsersSetUp: []SourceUser{
				aliceAzure,
				bobAzure,
				carolAzure,
			},
			ytUsersSetUp: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			ytUsersExpected: []YtsaurusUser{
				aliceYtsaurus,
				bobYtsaurus,
				carolYtsaurus,
			},
			ytGroupsSetUp: []YtsaurusGroupWithMembers{
				{
					YtsaurusGroup: devsYtsaurusGroup,
					Members: NewStringSetFromItems(
						aliceYtsaurus.Username,
						bobYtsaurus.Username,
					),
				},
				{
					YtsaurusGroup: hqYtsaurusGroup,
					Members: NewStringSetFromItems(
						aliceYtsaurus.Username,
						bobYtsaurus.Username,
					),
				},
			},
			azureGroupsSetUp: []SourceGroupWithMembers{
				{
					// This group should be updated.
					SourceGroup: devsAzureGroupChangedDisplayName,
					// Members list are also updated.
					Members: NewStringSetFromItems(
						aliceAzure.AzureID,
						carolAzure.AzureID,
					),
				},
				{
					// For this group only displayName should be updated.
					SourceGroup: hqAzureGroupChangedBackwardCompatible,
					// Members also changed.
					Members: NewStringSetFromItems(
						aliceAzure.AzureID,
						carolAzure.AzureID,
					),
				},
			},
			ytGroupsExpected: []YtsaurusGroupWithMembers{
				{
					YtsaurusGroup: devsYtsaurusGroupChangedDisplayName,
					Members: NewStringSetFromItems(
						aliceYtsaurus.Username,
						carolYtsaurus.Username,
					),
				},
				{
					YtsaurusGroup: hqYtsaurusGroupChangedBackwardCompatible,
					Members: NewStringSetFromItems(
						aliceYtsaurus.Username,
						carolYtsaurus.Username,
					),
				},
			},
		},
	}
)

// TestAppSync uses local YTsaurus container and fake Azure to test all the cases:
// [x] If Azure user not in YTsaurus -> created;
// [x] If Azure user already in YTsaurus no changes -> skipped;
// [x] If Azure user already in YTsaurus with changes -> updated;
// [x] If user in YTsaurus but not in Azure (and ban_before_remove_duration=0) -> removed;
// [x] If user in YTsaurus but not in Azure (and ban_before_remove_duration != 0) -> banned -> removed;
// [x] If Azure user without @azure attribute in YTsaurus —> ignored;
// [x] Azure user field updates is reflected in YTsaurus user;
// [x] YTsaurus username is built according to config;
// [x] YTsaurus username is in lowercase;
// [x] If Azure group is not exist in YTsaurus -> creating YTsaurus with members;
// [x] If YTsaurus group is not exist in Azure -> delete YTsaurus group;
// [x] If Azure group membership changed -> update in YTsaurus group membership;
// [x] If Azure group fields (though there are none extra fields) changed -> update YTsaurus group;
// [x] If Azure group displayName changed -> recreate YTsaurus group;
// [x] If Azure group displayName changed AND Azure members changed -> recreate YTsaurus group with actual members set;
// [x] YTsaurus group name is built according to config;
// [x] Remove limits config option works.
func TestAppSync(t *testing.T) {
	require.NoError(t, os.Setenv(defaultYtsaurusSecretEnvVar, ytDevToken))
	for _, tc := range testCases {
		t.Run(
			tc.name,
			func(tc testCase) func(t *testing.T) {
				return func(t *testing.T) {
					if tc.testTime.IsZero() {
						tc.testTime = initialTestTime
					}
					clock := testclock.NewFakePassiveClock(initialTestTime)

					ytLocal := NewYtsaurusLocal()
					defer func() { require.NoError(t, ytLocal.Stop()) }()
					require.NoError(t, ytLocal.Start())

					azure := NewAzureFake()
					azure.setUsers(tc.azureUsersSetUp)
					azure.setGroups(tc.azureGroupsSetUp)

					ytClient, err := ytLocal.GetClient()
					require.NoError(t, err)

					initialYtUsers, initialYtGroups := getAllYtsaurusObjects(t, ytClient)
					setupYtsaurusObjects(t, ytClient, tc.ytUsersSetUp, tc.ytGroupsSetUp)

					if tc.appConfig == nil {
						tc.appConfig = defaultAppConfig
					}
					app, err := NewAppCustomized(
						&Config{
							App:   *tc.appConfig,
							Azure: &AzureConfig{},
							Ytsaurus: YtsaurusConfig{
								Proxy:               ytLocal.GetProxy(),
								ApplyUserChanges:    true,
								ApplyGroupChanges:   true,
								ApplyMemberChanges:  true,
								LogLevel:            "DEBUG",
								SourceAttributeName: "azure",
							},
						}, getDevelopmentLogger(),
						azure,
						clock,
					)
					require.NoError(t, err)

					app.syncOnce()

					// we have eventually here, because user removal takes some time.
					require.Eventually(
						t,
						func() bool {
							udiff, gdiff := diffYtsaurusObjects(t, ytClient, tc.ytUsersExpected, initialYtUsers, tc.ytGroupsExpected, initialYtGroups)
							actualUsers, actualGroups := getAllYtsaurusObjects(t, ytClient)
							if udiff != "" {
								t.Log("Users diff is not empty yet:", udiff)
								t.Log("expected users", tc.ytUsersExpected)
								t.Log("actual users", actualUsers)
							}
							if gdiff != "" {
								t.Log("Groups diff is not empty yet:", gdiff)
								t.Log("expected groups", tc.ytGroupsExpected)
								t.Log("actual groups", actualGroups)
							}
							return udiff == "" && gdiff == ""
						},
						3*time.Second,
						300*time.Millisecond,
					)
				}
			}(tc),
		)
	}
}

func TestManageUnmanagedUsersIsForbidden(t *testing.T) {
	ytLocal := NewYtsaurusLocal()
	defer func() { require.NoError(t, ytLocal.Stop()) }()
	require.NoError(t, ytLocal.Start())

	ytClient, err := ytLocal.GetClient()
	require.NoError(t, err)

	ytsaurus, err := NewYtsaurus(
		&YtsaurusConfig{
			Proxy:    ytLocal.GetProxy(),
			LogLevel: "DEBUG",
		},
		getDevelopmentLogger(),
		testclock.NewFakePassiveClock(time.Now()),
	)
	require.NoError(t, err)

	unmanagedOleg := "oleg"

	err = doCreateYtsaurusUser(
		context.Background(),
		ytClient,
		unmanagedOleg,
		nil,
	)
	require.NoError(t, err)

	for _, username := range []string{
		"root",
		"guest",
		"job",
		unmanagedOleg,
	} {
		require.ErrorContains(t,
			ytsaurus.RemoveUser(username),
			"Prevented attempt to change manual managed user",
		)
		require.ErrorContains(t,
			ytsaurus.UpdateUser(username, YtsaurusUser{Username: username, SourceRaw: map[string]any{
				"email": "dummy@acme.com",
			}}),
			"Prevented attempt to change manual managed user",
		)
	}
}

func getAllYtsaurusObjects(t *testing.T, client yt.Client) (users []YtsaurusUser, groups []YtsaurusGroupWithMembers) {
	allUsers, err := doGetAllYtsaurusUsers(context.Background(), client, "azure")
	require.NoError(t, err)
	allGroups, err := doGetAllYtsaurusGroupsWithMembers(context.Background(), client, "azure")
	require.NoError(t, err)
	return allUsers, allGroups
}

func setupYtsaurusObjects(t *testing.T, client yt.Client, users []YtsaurusUser, groups []YtsaurusGroupWithMembers) {
	t.Log("Setting up yt for test")
	for _, user := range users {
		t.Logf("creating user: %v", user)
		err := doCreateYtsaurusUser(
			context.Background(),
			client,
			user.Username,
			buildUserAttributes(user, "azure"),
		)
		require.NoError(t, err)
	}

	for _, group := range groups {
		t.Log("creating group:", group)
		err := doCreateYtsaurusGroup(
			context.Background(),
			client,
			group.Name,
			buildGroupAttributes(group.YtsaurusGroup, "azure"),
		)
		require.NoError(t, err)
		for member := range group.Members.Iter() {
			err = doAddMemberYtsaurusGroup(
				context.Background(),
				client,
				member,
				group.Name,
			)
		}
		require.NoError(t, err)
	}
}

func diffYtsaurusObjects(t *testing.T, client yt.Client, expectedUsers, initialUsers []YtsaurusUser, expectedGroups, initalGroups []YtsaurusGroupWithMembers) (string, string) {
	actualUsers, actualGroups := getAllYtsaurusObjects(t, client)
	allExpectedUsers := append(initialUsers, expectedUsers...)
	allExpectedGroups := append(initalGroups, expectedGroups...)

	// It seems that `users`  group @members attr contains not the all users in the system:
	// for example it doesn't include:
	// alien_cell_synchronizer, file_cache, guest, operations_cleaner, operations_client, etc...
	// we don't want to test that.
	// Though we expect it to include users created in test, so we update group members in out expected group list.
	var expectedNewUsernamesInUsersGroup []string
	for _, u := range expectedUsers {
		expectedNewUsernamesInUsersGroup = append(expectedNewUsernamesInUsersGroup, u.Username)
	}
	for idx, group := range allExpectedGroups {
		if group.Name == "users" {
			for _, uname := range expectedNewUsernamesInUsersGroup {
				allExpectedGroups[idx].Members.Add(uname)
			}
		}
	}

	uDiff := cmp.Diff(
		actualUsers,
		allExpectedUsers,
		cmpopts.SortSlices(func(left, right YtsaurusUser) bool {
			return left.Username < right.Username
		}),
	)
	gDiff := cmp.Diff(
		actualGroups,
		allExpectedGroups,
		cmpopts.SortSlices(func(left, right YtsaurusGroupWithMembers) bool {
			return left.Name < right.Name
		}),
	)

	return uDiff, gDiff
}

func parseAppTime(timStr string) time.Time {
	parsed, err := time.Parse(appTimeFormat, timStr)
	if err != nil {
		panic("parsing " + timStr + " failed")
	}
	return parsed
}
