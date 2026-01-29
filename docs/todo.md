__________________________________________
#
# # # **IMPORTANT** # # # 


- The ctrl+k debug command that kills all enemy squads seems to kill every squad that is not in combat. Need to look at that

- calculateEntryIndexAtPosition in squadeditor_movesquad.go is not accurately calculating the position to move the squad to. Check if there is an ebitenui var (such as entryselected) you can use instead of calculating with pixel positions

- Check if we have to mark the caches as dirty, or whether the way the ecs library implements views handles that automatically

- When an encounter starts, we hide the encounter sprites for the combat sprites. Check to make sure that combat can't accidentally start if a combat squad moves into the position of a hidden encounter sprite 
* This is broken. Encounters can still start

- Handle Entity Cleanup




- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer

- Can always use the move undo command, even after attacking. That's a bug


- Creating a new squad breaks the threat map. Fix this


- Need to be able to toggle threat layer visualzation to player faction.






- Check if counter attack in ExecuteAttackAction accounts for dead units. Dead units should  not be part of the counter attack damage calculation
________________________________________________
# Testing 



# Cleanup

- Replace squad.ExecuteSquadAttack in simulator. 


- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 


- CombatController has a lot of artifacts of the old roguelike prior to the change. Specifically in regards to throwables. Find a way tohandle that. 


- Input Package has a lot of leftovers from the old roguelike




- combatqueries.go still has functions which search the entire ecs space



# Bug Fixes

- Throwables are completely broken. There is an out of bounds error


# Review

- Identify where there is a possiblity of cache invalidation errors. We started to cache things used for ECS query, so we need to 



- The Inventory is a leftover prior to shifting the game to a turn based squad tactics game from a regular roguelike. Think of what you want to do with the inventory

- Look at all of the const. See which ones should be defined in a file




- behavor and ai package


- Look into GameBootStrap


- Look at keysToTrack

- Review enounter and overworld package

# Additions 


- Add Abilities. Once they are, see how they are called in combat. Detemrine how you want to call them
- Better Combat Log
- Add squad equipment




# Redundancies?

- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead
- Check on ENABLE_COMBAT_LOG and ENABLE_COMBAT_LOG_EXPORT. Probabably some duplication we can get rid of