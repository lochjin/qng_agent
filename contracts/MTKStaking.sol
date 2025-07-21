// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract MTKStaking is ReentrancyGuard, Ownable {
    IERC20 public stakingToken; // MTK代币
    
    // 质押信息结构
    struct StakingInfo {
        uint256 amount;           // 质押数量
        uint256 startTime;        // 质押开始时间
        uint256 lastClaimTime;    // 上次领取奖励时间
        uint256 totalRewards;     // 累计奖励
    }
    
    // 用户质押信息
    mapping(address => StakingInfo) public stakingInfo;
    
    // 合约状态
    uint256 public totalStaked;                    // 总质押量
    uint256 public rewardRate = 100;               // 奖励率：每天每MTK获得100wei奖励
    uint256 public constant REWARD_PRECISION = 1e18; // 奖励精度
    uint256 public constant SECONDS_PER_DAY = 86400; // 一天的秒数
    
    // 最小质押数量
    uint256 public minStakeAmount = 1e18; // 1 MTK
    
    // 事件
    event Staked(address indexed user, uint256 amount);
    event Unstaked(address indexed user, uint256 amount);
    event RewardsClaimed(address indexed user, uint256 amount);
    event RewardRateUpdated(uint256 newRate);
    
    constructor(address _stakingToken) Ownable(msg.sender) {
        stakingToken = IERC20(_stakingToken);
    }
    
    // 质押MTK代币
    function stake(uint256 amount) external nonReentrant {
        require(amount >= minStakeAmount, "Amount below minimum stake");
        require(amount > 0, "Cannot stake 0 tokens");
        
        // 先领取之前的奖励
        if (stakingInfo[msg.sender].amount > 0) {
            _claimRewards();
        }
        
        // 转移代币到合约
        require(
            stakingToken.transferFrom(msg.sender, address(this), amount),
            "Transfer failed"
        );
        
        // 更新质押信息
        StakingInfo storage info = stakingInfo[msg.sender];
        info.amount += amount;
        info.startTime = block.timestamp;
        info.lastClaimTime = block.timestamp;
        
        // 更新总质押量
        totalStaked += amount;
        
        emit Staked(msg.sender, amount);
    }
    
    // 取消质押
    function unstake(uint256 amount) external nonReentrant {
        StakingInfo storage info = stakingInfo[msg.sender];
        require(info.amount >= amount, "Insufficient staked amount");
        require(amount > 0, "Cannot unstake 0 tokens");
        
        // 先领取奖励
        _claimRewards();
        
        // 更新质押信息
        info.amount -= amount;
        
        // 更新总质押量
        totalStaked -= amount;
        
        // 转移代币回用户
        require(stakingToken.transfer(msg.sender, amount), "Transfer failed");
        
        emit Unstaked(msg.sender, amount);
    }
    
    // 领取奖励
    function claimRewards() external nonReentrant {
        _claimRewards();
    }
    
    // 内部领取奖励函数
    function _claimRewards() internal {
        StakingInfo storage info = stakingInfo[msg.sender];
        require(info.amount > 0, "No staked tokens");
        
        uint256 reward = calculateReward(msg.sender);
        if (reward > 0) {
            info.totalRewards += reward;
            info.lastClaimTime = block.timestamp;
            
            // 这里简化为直接铸造奖励代币给用户
            // 实际项目中可能需要从奖励池转移或铸造新代币
            require(stakingToken.transfer(msg.sender, reward), "Reward transfer failed");
            
            emit RewardsClaimed(msg.sender, reward);
        }
    }
    
    // 计算用户当前可领取的奖励
    function calculateReward(address user) public view returns (uint256) {
        StakingInfo memory info = stakingInfo[user];
        if (info.amount == 0) {
            return 0;
        }
        
        uint256 timeStaked = block.timestamp - info.lastClaimTime;
        uint256 reward = (info.amount * rewardRate * timeStaked) / (SECONDS_PER_DAY * REWARD_PRECISION);
        
        return reward;
    }
    
    // 获取用户质押信息
    function getUserStakingInfo(address user) external view returns (
        uint256 stakedAmount,
        uint256 startTime,
        uint256 lastClaimTime,
        uint256 totalRewards,
        uint256 pendingRewards
    ) {
        StakingInfo memory info = stakingInfo[user];
        return (
            info.amount,
            info.startTime,
            info.lastClaimTime,
            info.totalRewards,
            calculateReward(user)
        );
    }
    
    // 获取用户质押余额
    function balanceOf(address user) external view returns (uint256) {
        return stakingInfo[user].amount;
    }
    
    // 管理员函数：设置奖励率
    function setRewardRate(uint256 newRate) external onlyOwner {
        rewardRate = newRate;
        emit RewardRateUpdated(newRate);
    }
    
    // 管理员函数：设置最小质押数量
    function setMinStakeAmount(uint256 newAmount) external onlyOwner {
        minStakeAmount = newAmount;
    }
    
    // 管理员函数：存入奖励代币
    function depositRewards(uint256 amount) external onlyOwner {
        require(
            stakingToken.transferFrom(msg.sender, address(this), amount),
            "Deposit failed"
        );
    }
    
    // 紧急提取函数（仅管理员）
    function emergencyWithdraw() external onlyOwner {
        uint256 balance = stakingToken.balanceOf(address(this));
        require(stakingToken.transfer(owner(), balance), "Emergency withdraw failed");
    }
    
    // 获取合约代币余额
    function getContractBalance() external view returns (uint256) {
        return stakingToken.balanceOf(address(this));
    }
} 