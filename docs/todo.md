__________________________________________
#
# # # **IMPORTANT** # # # 


- Checks if ECS related caches are really needed. To a performance profile before and after using Views rather than the custom cache

- GetUnitIDsInSquad takes too much time with a large number squads

- Need better node management system. Non-combat nodes, player nodes, and POIs in the overworld map gen are too separate. Different ways to do the same thigns

- Consider using an interface for intent in overworld

- Need to be able to equip items to squads

-  ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  

- Reneable victory conditions in tickmanager.go. They are disabled for testing

- encounter service should not have mode coordinator. It overlaps too much with what the GUI is doing


- calculateEntryIndexAtPosition in squadeditor_movesquad.go is not accurately calculating the position to move the squad to. Check if there is an ebitenui var (such as entryselected) you can use instead of calculating with pixel positions

- Check if we have to mark the caches as dirty, or whether the way the ecs library implements views handles that automatically

- Overworld needs difficulty setting

- Handle Entity Cleanup

- config.go in the Overworld package requires error handling in case there is an issue with the JSON file. Also add error handling for every other case of JSON loading


- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer

- Can always use the move undo command, even after attacking. That's a bug


- Creating a new squad breaks the threat map. Fix this


- Need to be able to toggle threat layer visualzation to player faction.

- Enemy squads appear to attack when they're out of range

- Encounted difficulty should partially be based on overworld stuff

- Fully explor encoutner types from encounterdata.json - determine how squad composition works in encounters




- Check if counter attack in ExecuteAttackAction accounts for dead units. Dead units should  not be part of the counter attack damage calculation
________________________________________________
# Testing 



# Cleanup


- Need a better job of highlighting was data comes from JSON files
- The way the GUI handles keys to track and keyboard presses need to improve



- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 


- CombatController has a lot of artifacts of the old roguelike prior to the change. Specifically in regards to throwables. Find a way tohandle that. 


- Input Package has a lot of leftovers from the old roguelike

- Review anything that was done to deal with circular dependencies


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