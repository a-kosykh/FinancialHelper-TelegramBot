package storage

/*
func Test_OnNewUser_IsUserAdded(t *testing.T) {
	storage := New()
	want := true
	got := storage.InitUser(123, false)

	if got != want {
		t.Errorf("Initing user failed")
	}
}

func Test_OnReset_ResetData(t *testing.T) {
	storage := New()
	userData := userdata.UserData{
		ExpencesMap:  make(map[string][]expences.Expence),
		BaseCurrency: "RUB",
	}
	userData.ExpencesMap["food"] = append(userData.ExpencesMap["food"], expences.Expence{
		Total:     100,
		Timestamp: time.Now(),
	})

	storage.Table[123] = userData
	storage.InitUser(123, true)

	got := len(storage.Table[123].ExpencesMap)
	want := 0

	if got != want {
		t.Errorf("User data failed to reset")
	}
}
*/
