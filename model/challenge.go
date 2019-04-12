package model

import (
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
)

//Challenge a Challenge Record
type Challenge struct {
	ID            string `gorm:"primary_key"`
	Genre         string
	Name          string
	AuthorID      string
	Author        *User `gorm:"foreignkey:AuthorID"`
	Score         int
	Caption       string `sql:"type:varchar(1500);"`
	Hints         []*Hint
	Flags         []*Flag
	Answer        string
	WhoSolved     []*User `gorm:"many2many:user_solved_challenges;"`
	WhoChallenged []*User `gorm:"many2many:user_challenged_challenges;"`
	Votes         []*Vote
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

//Flag a Flag Record
type Flag struct {
	ID          string `gorm:"primary_key"`
	ChallengeID string
	Flag        string
	Score       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

//Hint a Hint Record
type Hint struct {
	ID          string `gorm:"primary_key"`
	ChallengeID string
	Caption     string `sql:"type:varchar(3000);"`
	Penalty     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

//Vote a Vote Record
type Vote struct {
	gorm.Model
	ChallengeID string `gorm:"unique_index:idx_challenge_id_user_id"`
	UserID      string `gorm:"unique_index:idx_challenge_id_user_id"`
	Vote        string
}

//ErrChallengeNotFound an Error due to the Challenge Not Found
var ErrChallengeNotFound = gorm.ErrRecordNotFound

//GetChallenges Get All Challenge Records
func GetChallenges() ([]*Challenge, error) {
	challenges := make([]*Challenge, 0)
	if err := db.Preload("Author").Preload("Hints").Preload("Flags").Preload("WhoSolved").Preload("WhoChallenged").Preload("Votes").Find(&challenges).Error; err != nil {
		return nil, err
	}
	return challenges, nil
}

//GetChallengeByID Get the Challenge Record by its ID
func GetChallengeByID(id string) (*Challenge, error) {
	challenge := &Challenge{}
	if err := db.Where(&Challenge{ID: id}).Preload("Author").Preload("Hints").Preload("Flags").Preload("WhoSolved").Preload("WhoChallenged").Preload("Votes").First(challenge).Error; err != nil {
		return nil, err
	}
	return challenge, nil
}

//NewChallenge Make a New Challenge Record
func NewChallenge(genre string, name string, authorID string, score int, caption string, captions []string, penalties []int, flags []string, scores []int, answer string) (*Challenge, error) {
	id := uuid.NewV4().String()
	hints := make([]*Hint, len(captions))
	for i := 0; i < len(captions); i++ {
		hints[i] = &Hint{
			ID:      id + ":" + strconv.Itoa(i),
			Caption: captions[i],
			Penalty: penalties[i],
		}
	}
	_flags := make([]*Flag, len(flags))
	for i := 0; i < len(flags); i++ {
		_flags[i] = &Flag{
			ID:    id + ":" + strconv.Itoa(i),
			Flag:  flags[i],
			Score: scores[i],
		}
	}

	tx := db.Begin()

	author := &User{}
	if err := tx.Where(&User{ID: authorID}).First(author).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	challenge := &Challenge{
		ID:      id,
		Genre:   genre,
		Name:    name,
		Author:  author,
		Score:   score,
		Caption: caption,
		Hints:   hints,
		Flags:   _flags,
		Answer:  answer,
	}
	if err := tx.Set("gorm:save_associations", true).Create(challenge).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return challenge, tx.Commit().Error
}

//Delete Delete the Challenge Record
func (challenge *Challenge) Delete() error {
	return db.Delete(challenge).Error
}

//Update Update the Challenge Record
func (challenge *Challenge) Update(genre string, name string, authorID string, score int, caption string, captions []string, penalties []int, flags []string, scores []int, answer string) error {
	hints := make([]*Hint, len(captions))
	for i := 0; i < len(captions); i++ {
		hints[i] = &Hint{
			ID:      challenge.ID + ":" + strconv.Itoa(i),
			Caption: captions[i],
			Penalty: penalties[i],
		}
	}
	_flags := make([]*Flag, len(flags))
	for i := 0; i < len(flags); i++ {
		_flags[i] = &Flag{
			ID:    challenge.ID + ":" + strconv.Itoa(i),
			Flag:  flags[i],
			Score: scores[i],
		}
	}

	tx := db.Begin()

	author := &User{}
	if err := tx.Where(&User{ID: authorID}).First(author).Error; err != nil {
		tx.Rollback()
		return err
	}

	challenge.Genre, challenge.Name, challenge.Author, challenge.Score, challenge.Caption, challenge.Hints, challenge.Flags, challenge.Answer = genre, name, author, score, caption, hints, _flags, answer
	if err := tx.Set("gorm:save_associations", true).Save(challenge).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (challenge *Challenge) calcScore(hints []*Hint, _flags []*Flag) int {
	_flagNum := -1
	for i := 0; i < len(_flags); i++ {
		idSplit := strings.Split(_flags[i].ID, ":")
		_flagNumTemp, _ := strconv.Atoi(idSplit[1])
		if _flagNum < _flagNumTemp {
			_flagNum = _flagNumTemp
		}
	}

	score := 0
	if _flagNum >= 0 {
		score += challenge.Flags[_flagNum].Score
	}
	for _, hint := range hints {
		score -= hint.Penalty
	}
	if score < 0 {
		score = 0
	}
	return score
}

//CheckAnswer Check the Answer
func (challenge *Challenge) CheckAnswer(user *User, flag string) (bool, int, error) {
	tx := db.Begin()

	hints := make([]*Hint, 0)
	if err := tx.Where(&Hint{ChallengeID: challenge.ID}).Model(user).Association("OpenedHints").Find(&hints).Error; err != nil {
		tx.Rollback()
		return false, 0, err
	}
	_flags := make([]*Flag, 0)
	if err := tx.Where(&Flag{ChallengeID: challenge.ID}).Model(user).Association("FoundFlags").Find(&_flags).Error; err != nil {
		tx.Rollback()
		return false, 0, err
	}

	isCorrect := true
	_flag := &Flag{}
	err := tx.Where(&Flag{ChallengeID: challenge.ID, Flag: flag}).First(_flag).Error
	if err == gorm.ErrRecordNotFound {
		isCorrect = false
	} else if err != nil {
		tx.Rollback()
		return false, 0, err
	}

	scoreDelta := 0
	if isCorrect {
		scoreDelta = challenge.calcScore(hints, append(_flags, _flag)) - challenge.calcScore(hints, _flags)
	}
	user.Score += scoreDelta

	if err := tx.Model(challenge).Association("WhoChallenged").Append(user).Error; err != nil {
		tx.Rollback()
		return false, 0, err
	}

	if isCorrect {
		idSplit := strings.Split(_flag.ID, ":")
		_flagNum, _ := strconv.Atoi(idSplit[1])

		if err := tx.Model(user).Association("FoundFlags").Append(challenge.Flags[_flagNum]).Error; err != nil {
			tx.Rollback()
			return false, 0, err
		}

		if _flagNum == len(challenge.Flags)-1 {
			if err := tx.Model(challenge).Association("WhoSolved").Append(user).Error; err != nil {
				tx.Rollback()
				return false, 0, err
			}
		}

		user.LastSolvedChallenge = challenge
		user.LastSolvedTime = time.Now()
	}

	if err := tx.Save(user).Error; err != nil {
		tx.Rollback()
		return false, 0, err
	}

	return isCorrect, scoreDelta, tx.Commit().Error
}

//GetVote Get the User's Vote for the Challenge
func (challenge *Challenge) GetVote(userID string) (string, error) {
	_vote := &Vote{}
	if err := db.Where(&Vote{ChallengeID: challenge.ID, UserID: userID}).First(_vote).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return _vote.Vote, nil
}

//PutVote Put the User's Vote for the Challenge
func (challenge *Challenge) PutVote(userID string, vote string) error {
	_vote := &Vote{
		ChallengeID: challenge.ID,
		UserID:      userID,
		Vote:        vote,
	}
	return db.Create(_vote).Error
}
