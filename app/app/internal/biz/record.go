package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"strconv"
	"strings"
	"time"
)

type EthUserRecord struct {
	ID        int64
	UserId    int64
	Hash      string
	Status    string
	Type      string
	Amount    string
	RelAmount int64
	CoinType  string
}

type Location struct {
	ID           int64
	UserId       int64
	Status       string
	CurrentLevel int64
	Current      int64
	CurrentMax   int64
	Row          int64
	Col          int64
	StopDate     time.Time
	CreatedAt    time.Time
}

type LocationNew struct {
	ID                int64
	UserId            int64
	Status            string
	Current           int64
	CurrentMax        int64
	StopLocationAgain int64
	OutRate           int64
	StopCoin          int64
	StopDate          time.Time
	CreatedAt         time.Time
}

type GlobalLock struct {
	ID     int64
	Status int64
}

type RecordUseCase struct {
	ethUserRecordRepo             EthUserRecordRepo
	userRecommendRepo             UserRecommendRepo
	configRepo                    ConfigRepo
	locationRepo                  LocationRepo
	userBalanceRepo               UserBalanceRepo
	userInfoRepo                  UserInfoRepo
	userCurrentMonthRecommendRepo UserCurrentMonthRecommendRepo
	tx                            Transaction
	log                           *log.Helper
}

type EthUserRecordRepo interface {
	GetEthUserRecordListByHash(ctx context.Context, hash ...string) (map[string]*EthUserRecord, error)
	CreateEthUserRecordListByHash(ctx context.Context, r *EthUserRecord) (*EthUserRecord, error)
}

type LocationRepo interface {
	CreateLocation(ctx context.Context, rel *Location) (*Location, error)
	GetLocationLast(ctx context.Context) (*Location, error)
	GetMyLocationLast(ctx context.Context, userId int64) (*Location, error)
	GetLocationDailyYesterday(ctx context.Context, day int) ([]*LocationNew, error)
	GetMyStopLocationLast(ctx context.Context, userId int64) (*Location, error)
	GetMyLocationRunningLast(ctx context.Context, userId int64) (*Location, error)
	GetLocationsByUserId(ctx context.Context, userId int64) ([]*Location, error)
	GetRewardLocationByRowOrCol(ctx context.Context, row int64, col int64, locationRowConfig int64) ([]*Location, error)
	GetRewardLocationByIds(ctx context.Context, ids ...int64) (map[int64]*Location, error)
	UpdateLocation(ctx context.Context, id int64, status string, current int64, stopDate time.Time) error
	GetLocations(ctx context.Context, b *Pagination, userId int64) ([]*LocationNew, error, int64)
	GetLocationsAll(ctx context.Context, b *Pagination, userId int64) ([]*LocationNew, error, int64)
	UpdateLocationRowAndCol(ctx context.Context, id int64) error
	GetLocationsStopNotUpdate(ctx context.Context) ([]*Location, error)
	LockGlobalLocation(ctx context.Context) (bool, error)
	UnLockGlobalLocation(ctx context.Context) (bool, error)
	LockGlobalWithdraw(ctx context.Context) (bool, error)
	UnLockGlobalWithdraw(ctx context.Context) (bool, error)
	GetLockGlobalLocation(ctx context.Context) (*GlobalLock, error)
	GetLocationUserCount(ctx context.Context) int64
	GetLocationByIds(ctx context.Context, userIds ...int64) ([]*Location, error)
	GetAllLocations(ctx context.Context) ([]*Location, error)
	GetLocationsByUserIds(ctx context.Context, userIds []int64) ([]*Location, error)

	CreateLocationNew(ctx context.Context, rel *LocationNew) (*LocationNew, error)
	GetMyStopLocationsLast(ctx context.Context, userId int64) ([]*LocationNew, error)
	GetLocationsNewByUserId(ctx context.Context, userId int64) ([]*LocationNew, error)
	UpdateLocationNew(ctx context.Context, id int64, status string, current int64, stopDate time.Time, stopCoin int64) error
	GetRunningLocations(ctx context.Context) ([]*LocationNew, error)
}

func NewRecordUseCase(
	ethUserRecordRepo EthUserRecordRepo,
	locationRepo LocationRepo,
	userBalanceRepo UserBalanceRepo,
	userRecommendRepo UserRecommendRepo,
	userInfoRepo UserInfoRepo,
	configRepo ConfigRepo,
	userCurrentMonthRecommendRepo UserCurrentMonthRecommendRepo,
	tx Transaction,
	logger log.Logger) *RecordUseCase {
	return &RecordUseCase{
		ethUserRecordRepo:             ethUserRecordRepo,
		locationRepo:                  locationRepo,
		configRepo:                    configRepo,
		userRecommendRepo:             userRecommendRepo,
		userBalanceRepo:               userBalanceRepo,
		userCurrentMonthRecommendRepo: userCurrentMonthRecommendRepo,
		userInfoRepo:                  userInfoRepo,
		tx:                            tx,
		log:                           log.NewHelper(logger),
	}
}

func (ruc *RecordUseCase) GetEthUserRecordByTxHash(ctx context.Context, txHash ...string) (map[string]*EthUserRecord, error) {
	return ruc.ethUserRecordRepo.GetEthUserRecordListByHash(ctx, txHash...)
}

func (ruc *RecordUseCase) EthUserRecordHandle(ctx context.Context, ethUserRecord ...*EthUserRecord) (bool, error) {

	var (
		configs        []*Config
		recommendNeed  int64
		timeAgain      int64
		outRate        int64
		rewardRate     int64
		coinPrice      int64
		coinRewardRate int64
	)
	// 配置
	configs, _ = ruc.configRepo.GetConfigByKeys(ctx, "recommend_need", "time_again", "out_rate", "coin_price", "reward_rate", "coin_reward_rate")
	if nil != configs {
		for _, vConfig := range configs {
			if "recommend_need" == vConfig.KeyName {
				recommendNeed, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			} else if "time_again" == vConfig.KeyName {
				timeAgain, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			} else if "out_rate" == vConfig.KeyName {
				outRate, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			} else if "coin_price" == vConfig.KeyName {
				coinPrice, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			} else if "coin_reward_rate" == vConfig.KeyName {
				coinRewardRate, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			} else if "reward_rate" == vConfig.KeyName {
				rewardRate, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			}
		}
	}

	for _, v := range ethUserRecord {
		fmt.Println(v)
		var (
			currentValue                     int64
			amount                           int64
			locationCurrent                  int64
			stopCoin                         int64
			locationCurrentMax               int64
			currentLocationNew               *LocationNew
			userRecommend                    *UserRecommend
			myUserRecommendUserId            int64
			myUserRecommendUserInfo          *UserInfo
			myUserRecommendUserLocationsLast []*LocationNew
			myLastStopLocations              []*LocationNew
			tmpRecommendUserIds              []string
			dhbAmount                        int64
			err                              error
		)

		// 金额
		locationCurrentMax = v.RelAmount * outRate
		currentValue = v.RelAmount
		amount = v.RelAmount

		// 推荐人
		userRecommend, err = ruc.userRecommendRepo.GetUserRecommendByUserId(ctx, v.UserId)
		if nil != err {
			continue
		}
		if "" != userRecommend.RecommendCode {
			tmpRecommendUserIds = strings.Split(userRecommend.RecommendCode, "D")
			if 2 <= len(tmpRecommendUserIds) {
				myUserRecommendUserId, _ = strconv.ParseInt(tmpRecommendUserIds[len(tmpRecommendUserIds)-1], 10, 64) // 最后一位是直推人
			}
		}
		if 0 < myUserRecommendUserId {
			myUserRecommendUserInfo, err = ruc.userInfoRepo.GetUserInfoByUserId(ctx, myUserRecommendUserId)
		}

		// 冻结
		myLastStopLocations, err = ruc.locationRepo.GetMyStopLocationsLast(ctx, v.UserId)
		now := time.Now().UTC().Add(8 * time.Hour)
		if nil != myLastStopLocations {
			for _, vMyLastStopLocations := range myLastStopLocations {
				if now.Before(vMyLastStopLocations.StopDate.Add(time.Duration(timeAgain) * time.Minute)) {
					locationCurrent += vMyLastStopLocations.Current - vMyLastStopLocations.CurrentMax // 补上
					stopCoin += vMyLastStopLocations.StopCoin
				}
			}
		}

		if err = ruc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
			tmpLocationStatus := "running"
			var tmpStopDate time.Time
			if locationCurrent >= locationCurrentMax {
				tmpLocationStatus = "stop"
				tmpStopDate = time.Now().UTC().Add(8 * time.Hour)
			}
			currentLocationNew, err = ruc.locationRepo.CreateLocationNew(ctx, &LocationNew{ // 占位
				UserId:     v.UserId,
				Status:     tmpLocationStatus,
				Current:    locationCurrent,
				CurrentMax: locationCurrentMax,
				OutRate:    outRate,
				StopDate:   tmpStopDate,
			})
			if nil != err {
				return err
			}

			// 推荐人
			if nil != myUserRecommendUserInfo {
				// 有占位信息，推荐人第一代
				myUserRecommendUserLocationsLast, err = ruc.locationRepo.GetLocationsNewByUserId(ctx, myUserRecommendUserInfo.UserId)
				if nil != myUserRecommendUserLocationsLast {
					var myUserRecommendUserLocationLast *LocationNew
					if 1 <= len(myUserRecommendUserLocationsLast) {
						myUserRecommendUserLocationLast = myUserRecommendUserLocationsLast[0]
						for _, vMyUserRecommendUserLocationLast := range myUserRecommendUserLocationsLast {
							if "running" == vMyUserRecommendUserLocationLast.Status {
								myUserRecommendUserLocationLast = vMyUserRecommendUserLocationLast
								break
							}
						}

						tmpStatus := myUserRecommendUserLocationLast.Status // 现在还在运行中

						// 奖励usdt
						tmpBalanceAmount := myUserRecommendUserLocationLast.CurrentMax / myUserRecommendUserLocationLast.OutRate / 100 * recommendNeed / 100 * rewardRate // 记录下一次
						// 奖励币
						tmpBalanceCoinAmount := myUserRecommendUserLocationLast.CurrentMax / myUserRecommendUserLocationLast.OutRate / 100 * recommendNeed / 100 * coinRewardRate / 1000 * coinPrice

						myUserRecommendUserLocationLast.Status = "running"
						myUserRecommendUserLocationLast.Current += tmpBalanceAmount
						if myUserRecommendUserLocationLast.Current >= myUserRecommendUserLocationLast.CurrentMax { // 占位分红人分满停止
							myUserRecommendUserLocationLast.Status = "stop"
							if "running" == tmpStatus {
								myUserRecommendUserLocationLast.StopDate = time.Now().UTC().Add(8 * time.Hour)
							}
						}
						if 0 < tmpBalanceAmount {
							err = ruc.locationRepo.UpdateLocationNew(ctx, myUserRecommendUserLocationLast.ID, myUserRecommendUserLocationLast.Status, tmpBalanceAmount, myUserRecommendUserLocationLast.StopDate, tmpBalanceCoinAmount) // 分红占位数据修改
							if nil != err {
								return err
							}
						}
						amount -= tmpBalanceAmount // 扣除

						if 0 < tmpBalanceAmount { // 这次还能分红
							_, err = ruc.userBalanceRepo.NormalRecommendReward(ctx, myUserRecommendUserId, tmpBalanceAmount, tmpBalanceCoinAmount, currentLocationNew.ID, tmpStatus) // 直推人奖励
							if nil != err {
								return err
							}

						}
					}

				}
			}

			// 修改用户推荐人区数据，修改自身区数据
			_, err = ruc.userRecommendRepo.UpdateUserAreaSelfAmount(ctx, v.UserId, currentValue/100000)
			if nil != err {
				return err
			}
			for _, vTmpRecommendUserIds := range tmpRecommendUserIds {
				vTmpRecommendUserId, _ := strconv.ParseInt(vTmpRecommendUserIds, 10, 64)
				if vTmpRecommendUserId > 0 {
					_, err = ruc.userRecommendRepo.UpdateUserAreaAmount(ctx, vTmpRecommendUserId, currentValue/100000)
					if nil != err {
						return err
					}
				}
			}

			_, err = ruc.userBalanceRepo.Deposit(ctx, v.UserId, currentValue, dhbAmount) // 充值
			if nil != err {
				return err
			}

			// 清算冻结
			if 0 < locationCurrent && nil != myLastStopLocations {
				var tmpCurrentAmount int64
				if locationCurrent > locationCurrentMax {
					tmpCurrentAmount = locationCurrentMax
				} else {
					tmpCurrentAmount = locationCurrent
				}
				_, err = ruc.userBalanceRepo.DepositLastNew(ctx, v.UserId, tmpCurrentAmount, stopCoin, myLastStopLocations) // 充值
				if nil != err {
					return err
				}
			}

			err = ruc.userBalanceRepo.SystemReward(ctx, amount, currentLocationNew.ID)
			if nil != err {
				return err
			}

			_, err = ruc.ethUserRecordRepo.CreateEthUserRecordListByHash(ctx, &EthUserRecord{
				Hash:     v.Hash,
				UserId:   v.UserId,
				Status:   v.Status,
				Type:     v.Type,
				Amount:   v.Amount,
				CoinType: v.CoinType,
			})
			if nil != err {
				return err
			}

			return nil
		}); nil != err {
			continue
		}
	}

	return true, nil
}

func (ruc *RecordUseCase) AdminLocationInsert(ctx context.Context, userId int64, amount int64) (bool, error) {

	var (
		lastLocation            *Location
		myLocations             []*Location
		locationCurrentLevel    int64
		locationCurrent         int64
		locationCurrentMax      int64
		locationRow             int64
		locationCol             int64
		currentLocation         *Location
		myLastStopLocation      *Location
		err                     error
		configs                 []*Config
		stopLocations           []*Location
		userRecommend           *UserRecommend
		tmpRecommendUserIds     []string
		myUserRecommendUserInfo *UserInfo
		myUserRecommendUserId   int64
		currentValue            int64
		timeAgain               int64
	)
	// 配置
	configs, _ = ruc.configRepo.GetConfigByKeys(ctx, "time_again")
	if nil != configs {
		for _, vConfig := range configs {
			if "time_again" == vConfig.KeyName {
				timeAgain, _ = strconv.ParseInt(vConfig.Value, 10, 64)
			}
		}
	}

	// 调整位置紧缩
	stopLocations, err = ruc.locationRepo.GetLocationsStopNotUpdate(ctx)
	if nil != stopLocations {
		// 调整位置紧缩
		for _, vStopLocations := range stopLocations {

			if err = ruc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
				err = ruc.locationRepo.UpdateLocationRowAndCol(ctx, vStopLocations.ID)
				if nil != err {
					return err
				}
				return nil
			}); nil != err {
				continue
			}
		}
	}

	// 获取当前用户的占位信息，已经有运行中的跳过
	myLocations, err = ruc.locationRepo.GetLocationsByUserId(ctx, userId)
	if nil == myLocations { // 查询异常跳过本次循环
		return false, errors.New(500, "ERROR", "查询错误，重试")
	}
	if 0 < len(myLocations) { // 也代表复投
		tmpStatusRunning := false
		for _, vMyLocations := range myLocations {
			if "running" == vMyLocations.Status {
				tmpStatusRunning = true
				break
			}
		}

		if tmpStatusRunning { // 有运行中直接跳过本次循环
			return false, errors.New(500, "ERROR", "已存在运行中位置信息")
		}
	}

	// 获取最后一行数据
	lastLocation, err = ruc.locationRepo.GetLocationLast(ctx)
	if nil == lastLocation {
		locationRow = 1
		locationCol = 1
		fmt.Println(25, locationRow, locationRow)
	} else {
		if 3 > lastLocation.Col {
			locationCol = lastLocation.Col + 1
			locationRow = lastLocation.Row
			fmt.Println(33, locationCol, locationRow)
		} else {
			locationCol = 1
			locationRow = lastLocation.Row + 1
			fmt.Println(22, locationRow, locationRow)
		}
	}

	// todo
	if 50 == amount {
		locationCurrentLevel = 1
		locationCurrentMax = 5000000000000
		currentValue = 1000000000000
	} else if 100 == amount {
		locationCurrentLevel = 2
		locationCurrentMax = 15000000000000
		currentValue = 3000000000000
	} else if 300 == amount {
		locationCurrentLevel = 3
		locationCurrentMax = 25000000000000
		currentValue = 5000000000000
	} else {
		return false, errors.New(500, "ERROR", "输入金额错误，重试")
	}

	// 冻结
	myLastStopLocation, err = ruc.locationRepo.GetMyStopLocationLast(ctx, userId)
	now := time.Now().UTC().Add(8 * time.Hour)
	if nil != myLastStopLocation && now.Before(myLastStopLocation.StopDate.Add(time.Duration(timeAgain)*time.Minute)) {
		locationCurrent = myLastStopLocation.Current - myLastStopLocation.CurrentMax // 补上
	}

	// 推荐人
	userRecommend, err = ruc.userRecommendRepo.GetUserRecommendByUserId(ctx, userId)
	if nil != err {
		return false, errors.New(500, "ERROR", "输入金额错误，重试")
	}
	if "" != userRecommend.RecommendCode {
		tmpRecommendUserIds = strings.Split(userRecommend.RecommendCode, "D")
		if 2 <= len(tmpRecommendUserIds) {
			myUserRecommendUserId, _ = strconv.ParseInt(tmpRecommendUserIds[len(tmpRecommendUserIds)-1], 10, 64) // 最后一位是直推人
		}
	}

	if 0 < myUserRecommendUserId {
		myUserRecommendUserInfo, err = ruc.userInfoRepo.GetUserInfoByUserId(ctx, myUserRecommendUserId)
	}
	// 推荐人
	if nil != myUserRecommendUserInfo {
		if 0 == len(myLocations) { // vip 等级调整，被推荐人首次入单
			myUserRecommendUserInfo.HistoryRecommend += 1
			if myUserRecommendUserInfo.HistoryRecommend >= 10 {
				myUserRecommendUserInfo.Vip = 5
			} else if myUserRecommendUserInfo.HistoryRecommend >= 8 {
				myUserRecommendUserInfo.Vip = 4
			} else if myUserRecommendUserInfo.HistoryRecommend >= 6 {
				myUserRecommendUserInfo.Vip = 3
			} else if myUserRecommendUserInfo.HistoryRecommend >= 4 {
				myUserRecommendUserInfo.Vip = 2
			} else if myUserRecommendUserInfo.HistoryRecommend >= 2 {
				myUserRecommendUserInfo.Vip = 1
			}
		}
	}

	if err = ruc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		currentLocation, err = ruc.locationRepo.CreateLocation(ctx, &Location{ // 占位
			UserId:       userId,
			Status:       "running",
			CurrentLevel: locationCurrentLevel,
			Current:      locationCurrent,
			CurrentMax:   locationCurrentMax,
			Row:          locationRow,
			Col:          locationCol,
		})
		if nil != err {
			return err
		}

		_, err = ruc.userInfoRepo.UpdateUserInfo(ctx, myUserRecommendUserInfo) // 推荐人信息修改
		if nil != err {
			return err
		}

		_, err = ruc.userCurrentMonthRecommendRepo.CreateUserCurrentMonthRecommend(ctx, &UserCurrentMonthRecommend{ // 直推人本月推荐人数
			UserId:          myUserRecommendUserId,
			RecommendUserId: userId,
			Date:            time.Now().UTC().Add(8 * time.Hour),
		})
		if nil != err {
			return err
		}

		if 0 < locationCurrent && nil != myLastStopLocation {
			_, err = ruc.userBalanceRepo.DepositLast(ctx, userId, locationCurrent, myLastStopLocation.ID) // 充值
			if nil != err {
				return err
			}
		}

		// 修改用户推荐人区数据，修改自身区数据
		_, err = ruc.userRecommendRepo.UpdateUserAreaSelfAmount(ctx, userId, currentValue/10000000000)
		if nil != err {
			return err
		}
		for _, vTmpRecommendUserIds := range tmpRecommendUserIds {
			vTmpRecommendUserId, _ := strconv.ParseInt(vTmpRecommendUserIds, 10, 64)
			if vTmpRecommendUserId > 0 {
				_, err = ruc.userRecommendRepo.UpdateUserAreaAmount(ctx, vTmpRecommendUserId, currentValue/10000000000)
				if nil != err {
					return err
				}
			}
		}

		return nil
	}); nil != err {
		return false, errors.New(500, "ERROR", "错误，重试")

	}

	// 调整位置紧缩
	stopLocations, err = ruc.locationRepo.GetLocationsStopNotUpdate(ctx)
	if nil != stopLocations {
		// 调整位置紧缩
		for _, vStopLocations := range stopLocations {

			if err = ruc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
				err = ruc.locationRepo.UpdateLocationRowAndCol(ctx, vStopLocations.ID)
				if nil != err {
					return err
				}
				return nil
			}); nil != err {
				continue
			}
		}
	}

	return true, nil
}

func (ruc *RecordUseCase) LockEthUserRecordHandle(ctx context.Context, ethUserRecord ...*EthUserRecord) (bool, error) {
	var (
		lock bool
	)
	// todo 全局锁
	for i := 0; i < 3; i++ {
		lock, _ = ruc.locationRepo.LockGlobalLocation(ctx)
		if lock {
			return true, nil
		}
		time.Sleep(5 * time.Second)
	}

	return false, nil
}

func (ruc *RecordUseCase) UnLockEthUserRecordHandle(ctx context.Context, ethUserRecord ...*EthUserRecord) (bool, error) {
	return ruc.locationRepo.UnLockGlobalLocation(ctx)
}
